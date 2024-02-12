package pop

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gobuffalo/fizz"
	"github.com/gobuffalo/fizz/translators"
	"github.com/gobuffalo/pop/v6/columns"
	"github.com/gobuffalo/pop/v6/internal/defaults"
	"github.com/gobuffalo/pop/v6/logging"
	_ "github.com/jackc/pgx/v4/stdlib" // Import PostgreSQL driver
	"github.com/jmoiron/sqlx"
)

const nameCockroach = "cockroach"
const portCockroach = "26257"

const selectTablesQueryCockroach = "select table_name from information_schema.tables where table_schema = 'public' and table_type = 'BASE TABLE' and table_name <> ? and table_catalog = ?"
const selectTablesQueryCockroachV1 = "select table_name from information_schema.tables where table_name <> ? and table_schema = ?"

func init() {
	AvailableDialects = append(AvailableDialects, nameCockroach)
	dialectSynonyms["cockroachdb"] = nameCockroach
	dialectSynonyms["crdb"] = nameCockroach
	finalizer[nameCockroach] = finalizerCockroach
	newConnection[nameCockroach] = newCockroach
}

var _ dialect = &cockroach{}

// ServerInfo holds informational data about connected database server.
type cockroachInfo struct {
	VersionString string `db:"version"`
	product       string `db:"-"`
	license       string `db:"-"`
	version       string `db:"-"`
	buildInfo     string `db:"-"`
	client        string `db:"-"`
}

type cockroach struct {
	commonDialect
	translateCache map[string]string
	mu             sync.Mutex
	info           cockroachInfo
}

func (p *cockroach) Name() string {
	return nameCockroach
}

func (p *cockroach) DefaultDriver() string {
	return "pgx"
}

func (p *cockroach) Details() *ConnectionDetails {
	return p.ConnectionDetails
}

func (p *cockroach) Create(c *Connection, model *Model, cols columns.Columns) error {
	keyType, err := model.PrimaryKeyType()
	if err != nil {
		return err
	}
	switch keyType {
	case "int", "int64":
		cols.Remove(model.IDField())
		w := cols.Writeable()
		var query string
		if len(w.Cols) > 0 {
			query = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) returning %s", p.Quote(model.TableName()), w.QuotedString(p), w.SymbolizedString(), model.IDField())
		} else {
			query = fmt.Sprintf("INSERT INTO %s DEFAULT VALUES returning %s", p.Quote(model.TableName()), model.IDField())
		}
		txlog(logging.SQL, c, query, model.Value)
		stmt, err := c.Store.PrepareNamed(query)
		if err != nil {
			return err
		}
		id := map[string]interface{}{}
		err = stmt.QueryRow(model.Value).MapScan(id)
		if err != nil {
			if closeErr := stmt.Close(); closeErr != nil {
				return fmt.Errorf("failed to close prepared statement: %s: %w", closeErr, err)
			}
			return err
		}
		model.setID(id[model.IDField()])
		if err := stmt.Close(); err != nil {
			return fmt.Errorf("failed to close statement: %w", err)
		}
		return nil
	}
	return genericCreate(c, model, cols, p)
}

func (p *cockroach) Update(c *Connection, model *Model, cols columns.Columns) error {
	return genericUpdate(c, model, cols, p)
}

func (p *cockroach) UpdateQuery(c *Connection, model *Model, cols columns.Columns, query Query) (int64, error) {
	return genericUpdateQuery(c, model, cols, p, query, sqlx.DOLLAR)
}

func (p *cockroach) Destroy(c *Connection, model *Model) error {
	stmt := p.TranslateSQL(fmt.Sprintf("DELETE FROM %s AS %s WHERE %s", p.Quote(model.TableName()), model.Alias(), model.WhereID()))
	_, err := genericExec(c, stmt, model.ID())
	return err
}

func (p *cockroach) Delete(c *Connection, model *Model, query Query) error {
	return genericDelete(c, model, query)
}

func (p *cockroach) SelectOne(c *Connection, model *Model, query Query) error {
	return genericSelectOne(c, model, query)
}

func (p *cockroach) SelectMany(c *Connection, models *Model, query Query) error {
	return genericSelectMany(c, models, query)
}

func (p *cockroach) CreateDB() error {
	// createdb -h db -p 5432 -U cockroach enterprise_development
	deets := p.ConnectionDetails

	db, err := openPotentiallyInstrumentedConnection(p, p.urlWithoutDb())
	if err != nil {
		return fmt.Errorf("error creating Cockroach database %s: %w", deets.Database, err)
	}
	defer db.Close()
	query := fmt.Sprintf("CREATE DATABASE %s", p.Quote(deets.Database))
	log(logging.SQL, query)

	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating Cockroach database %s: %w", deets.Database, err)
	}

	log(logging.Info, "created database %s", deets.Database)
	return nil
}

func (p *cockroach) DropDB() error {
	deets := p.ConnectionDetails

	db, err := openPotentiallyInstrumentedConnection(p, p.urlWithoutDb())
	if err != nil {
		return fmt.Errorf("error dropping Cockroach database %s: %w", deets.Database, err)
	}
	defer db.Close()
	query := fmt.Sprintf("DROP DATABASE %s CASCADE;", p.Quote(deets.Database))
	log(logging.SQL, query)

	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("error dropping Cockroach database %s: %w", deets.Database, err)
	}

	log(logging.Info, "dropped database %s", deets.Database)
	return nil
}

func (p *cockroach) URL() string {
	c := p.ConnectionDetails
	if c.URL != "" {
		return c.URL
	}
	s := "postgres://%s:%s@%s:%s/%s?%s"
	return fmt.Sprintf(s, c.User, url.QueryEscape(c.Password), c.Host, c.Port, c.Database, c.OptionsString(""))
}

func (p *cockroach) urlWithoutDb() string {
	c := p.ConnectionDetails
	s := "postgres://%s:%s@%s:%s/?%s"
	return fmt.Sprintf(s, c.User, url.QueryEscape(c.Password), c.Host, c.Port, c.OptionsString(""))
}

func (p *cockroach) MigrationURL() string {
	return p.URL()
}

func (p *cockroach) TranslateSQL(sql string) string {
	defer p.mu.Unlock()
	p.mu.Lock()

	if csql, ok := p.translateCache[sql]; ok {
		return csql
	}
	csql := sqlx.Rebind(sqlx.DOLLAR, sql)

	p.translateCache[sql] = csql
	return csql
}

func (p *cockroach) FizzTranslator() fizz.Translator {
	return translators.NewCockroach(p.URL(), p.Details().Database)
}

func (p *cockroach) DumpSchema(w io.Writer) error {
	cmd := exec.Command("cockroach", "sql", "-e", "SHOW CREATE ALL TABLES", "-d", p.Details().Database, "--format", "raw")

	c := p.ConnectionDetails
	if defaults.String(c.option("sslmode"), "disable") == "disable" || strings.Contains(c.RawOptions, "sslmode=disable") {
		cmd.Args = append(cmd.Args, "--insecure")
	}
	return cockroachDumpSchema(p.Details(), cmd, w)
}

func cockroachDumpSchema(deets *ConnectionDetails, cmd *exec.Cmd, w io.Writer) error {
	log(logging.SQL, strings.Join(cmd.Args, " "))

	var bb bytes.Buffer

	cmd.Stdout = &bb
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	// --format raw returns comments prefixed with # which is invalid, so we make it a valid SQL comment.
	result := regexp.MustCompile("(?m)^#").ReplaceAll(bb.Bytes(), []byte("-- #"))

	if _, err := w.Write(result); err != nil {
		return err
	}

	x := bytes.TrimSpace(result)
	if len(x) == 0 {
		return fmt.Errorf("unable to dump schema for %s", deets.Database)
	}

	log(logging.Info, "dumped schema for %s", deets.Database)
	return nil
}

func (p *cockroach) LoadSchema(r io.Reader) error {
	return genericLoadSchema(p, r)
}

func (p *cockroach) TruncateAll(tx *Connection) error {
	type table struct {
		TableName string `db:"table_name"`
	}

	tableQuery := p.tablesQuery()

	var tables []table
	if err := tx.RawQuery(tableQuery, tx.MigrationTableName(), tx.Dialect.Details().Database).All(&tables); err != nil {
		return err
	}

	if len(tables) == 0 {
		return nil
	}

	tableNames := make([]string, len(tables))
	for i, t := range tables {
		tableNames[i] = t.TableName
		//! work around for current limitation of DDL and DML at the same transaction.
		//  it should be fixed when cockroach support it or with other approach.
		//  https://www.cockroachlabs.com/docs/stable/known-limitations.html#schema-changes-within-transactions
		if err := tx.RawQuery(fmt.Sprintf("delete from %s", p.Quote(t.TableName))).Exec(); err != nil {
			return err
		}
	}
	return nil
	// TODO!
	// return tx3.RawQuery(fmt.Sprintf("truncate %s cascade;", strings.Join(tableNames, ", "))).Exec()
}

func (p *cockroach) AfterOpen(c *Connection) error {
	if err := c.RawQuery(`select version() AS "version"`).First(&p.info); err != nil {
		return err
	}
	if s := strings.Split(p.info.VersionString, " "); len(s) > 3 {
		p.info.product = s[0]
		p.info.license = s[1]
		p.info.version = s[2]
		p.info.buildInfo = s[3]
	}
	log(logging.Debug, "server: %v %v %v", p.info.product, p.info.license, p.info.version)

	return nil
}

func newCockroach(deets *ConnectionDetails) (dialect, error) {
	deets.Dialect = "postgres"
	d := &cockroach{
		commonDialect:  commonDialect{ConnectionDetails: deets},
		translateCache: map[string]string{},
		mu:             sync.Mutex{},
	}
	d.info.client = deets.option("application_name")
	return d, nil
}

func finalizerCockroach(cd *ConnectionDetails) {
	appName := filepath.Base(os.Args[0])
	cd.setOptionWithDefault("application_name", cd.option("application_name"), appName)
	cd.Port = defaults.String(cd.Port, portCockroach)
	if cd.URL != "" {
		cd.URL = "postgres://" + trimCockroachPrefix(cd.URL)
	}
}

func trimCockroachPrefix(u string) string {
	parts := strings.Split(u, "://")
	if len(parts) != 2 {
		return u
	}
	return parts[1]
}

func (p *cockroach) tablesQuery() string {
	// See https://www.cockroachlabs.com/docs/stable/information-schema.html for more info about information schema changes
	tableQuery := selectTablesQueryCockroach
	if strings.HasPrefix(p.info.version, "v1.") {
		tableQuery = selectTablesQueryCockroachV1
	}
	return tableQuery
}
