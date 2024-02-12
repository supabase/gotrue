package pop

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	_mysql "github.com/go-sql-driver/mysql" // Load MySQL Go driver
	"github.com/gobuffalo/fizz"
	"github.com/gobuffalo/fizz/translators"
	"github.com/gobuffalo/pop/v6/columns"
	"github.com/gobuffalo/pop/v6/internal/defaults"
	"github.com/gobuffalo/pop/v6/logging"
	"github.com/jmoiron/sqlx"
)

const nameMySQL = "mysql"
const hostMySQL = "localhost"
const portMySQL = "3306"

func init() {
	AvailableDialects = append(AvailableDialects, nameMySQL)
	urlParser[nameMySQL] = urlParserMySQL
	finalizer[nameMySQL] = finalizerMySQL
	newConnection[nameMySQL] = newMySQL
}

var _ dialect = &mysql{}

type mysql struct {
	commonDialect
}

func (m *mysql) Name() string {
	return nameMySQL
}

func (m *mysql) DefaultDriver() string {
	return nameMySQL
}

func (mysql) Quote(key string) string {
	return fmt.Sprintf("`%s`", key)
}

func (m *mysql) Details() *ConnectionDetails {
	return m.ConnectionDetails
}

func (m *mysql) URL() string {
	cd := m.ConnectionDetails
	if cd.URL != "" {
		return strings.TrimPrefix(cd.URL, "mysql://")
	}

	user := fmt.Sprintf("%s:%s@", cd.User, cd.Password)
	user = strings.Replace(user, ":@", "@", 1)
	if user == "@" || strings.HasPrefix(user, ":") {
		user = ""
	}

	addr := fmt.Sprintf("(%s:%s)", cd.Host, cd.Port)
	// in case of unix domain socket, tricky.
	// it is better to check Host is not valid inet address or has '/'.
	if cd.Port == "socket" {
		addr = fmt.Sprintf("unix(%s)", cd.Host)
	}

	s := "%s%s/%s?%s"
	return fmt.Sprintf(s, user, addr, cd.Database, cd.OptionsString(""))
}

func (m *mysql) urlWithoutDb() string {
	cd := m.ConnectionDetails
	return strings.Replace(m.URL(), "/"+cd.Database+"?", "/?", 1)
}

func (m *mysql) MigrationURL() string {
	return m.URL()
}

func (m *mysql) Create(c *Connection, model *Model, cols columns.Columns) error {
	if err := genericCreate(c, model, cols, m); err != nil {
		return fmt.Errorf("mysql create: %w", err)
	}
	return nil
}

func (m *mysql) Update(c *Connection, model *Model, cols columns.Columns) error {
	if err := genericUpdate(c, model, cols, m); err != nil {
		return fmt.Errorf("mysql update: %w", err)
	}
	return nil
}

func (m *mysql) UpdateQuery(c *Connection, model *Model, cols columns.Columns, query Query) (int64, error) {
	if n, err := genericUpdateQuery(c, model, cols, m, query, sqlx.QUESTION); err != nil {
		return n, fmt.Errorf("mysql update query: %w", err)
	} else {
		return n, nil
	}
}

func (m *mysql) Destroy(c *Connection, model *Model) error {
	stmt := fmt.Sprintf("DELETE FROM %s  WHERE %s = ?", m.Quote(model.TableName()), model.IDField())
	_, err := genericExec(c, stmt, model.ID())
	if err != nil {
		return fmt.Errorf("mysql destroy: %w", err)
	}
	return nil
}

var asRegex = regexp.MustCompile(`\sAS\s\S+`) // exactly " AS non-spaces"

func (m *mysql) Delete(c *Connection, model *Model, query Query) error {
	sqlQuery, args := query.ToSQL(model)
	// * MySQL does not support table alias for DELETE syntax until 8.0.
	// * Do not generate SQL manually if they may have `WHERE IN`.
	// * Spaces are intentionally added to make it easy to see on the log.
	sqlQuery = asRegex.ReplaceAllString(sqlQuery, "  ")

	_, err := genericExec(c, sqlQuery, args...)
	return err
}

func (m *mysql) SelectOne(c *Connection, model *Model, query Query) error {
	if err := genericSelectOne(c, model, query); err != nil {
		return fmt.Errorf("mysql select one: %w", err)
	}
	return nil
}

func (m *mysql) SelectMany(c *Connection, models *Model, query Query) error {
	if err := genericSelectMany(c, models, query); err != nil {
		return fmt.Errorf("mysql select many: %w", err)
	}
	return nil
}

// CreateDB creates a new database, from the given connection credentials
func (m *mysql) CreateDB() error {
	deets := m.ConnectionDetails
	db, err := openPotentiallyInstrumentedConnection(m, m.urlWithoutDb())
	if err != nil {
		return fmt.Errorf("error creating MySQL database %s: %w", deets.Database, err)
	}
	defer db.Close()
	charset := defaults.String(deets.option("charset"), "utf8mb4")
	encoding := defaults.String(deets.option("collation"), "utf8mb4_general_ci")
	query := fmt.Sprintf("CREATE DATABASE `%s` DEFAULT CHARSET `%s` DEFAULT COLLATE `%s`", deets.Database, charset, encoding)
	log(logging.SQL, query)

	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("error creating MySQL database %s: %w", deets.Database, err)
	}

	log(logging.Info, "created database %s", deets.Database)
	return nil
}

// DropDB drops an existing database, from the given connection credentials
func (m *mysql) DropDB() error {
	deets := m.ConnectionDetails
	db, err := openPotentiallyInstrumentedConnection(m, m.urlWithoutDb())
	if err != nil {
		return fmt.Errorf("error dropping MySQL database %s: %w", deets.Database, err)
	}
	defer db.Close()
	query := fmt.Sprintf("DROP DATABASE `%s`", deets.Database)
	log(logging.SQL, query)

	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("error dropping MySQL database %s: %w", deets.Database, err)
	}

	log(logging.Info, "dropped database %s", deets.Database)
	return nil
}

func (m *mysql) TranslateSQL(sql string) string {
	return sql
}

func (m *mysql) FizzTranslator() fizz.Translator {
	t := translators.NewMySQL(m.URL(), m.Details().Database)
	return t
}

func (m *mysql) DumpSchema(w io.Writer) error {
	deets := m.Details()
	cmd := exec.Command("mysqldump", "-d", "-h", deets.Host, "-P", deets.Port, "-u", deets.User, fmt.Sprintf("--password=%s", deets.Password), deets.Database)
	if deets.Port == "socket" {
		cmd = exec.Command("mysqldump", "-d", "-S", deets.Host, "-u", deets.User, fmt.Sprintf("--password=%s", deets.Password), deets.Database)
	}
	return genericDumpSchema(deets, cmd, w)
}

// LoadSchema executes a schema sql file against the configured database.
func (m *mysql) LoadSchema(r io.Reader) error {
	return genericLoadSchema(m, r)
}

// TruncateAll truncates all tables for the given connection.
func (m *mysql) TruncateAll(tx *Connection) error {
	var stmts []string
	err := tx.RawQuery(mysqlTruncate, m.Details().Database, tx.MigrationTableName()).All(&stmts)
	if err != nil {
		return err
	}
	if len(stmts) == 0 {
		return nil
	}

	var qb bytes.Buffer
	// #49: Disable foreign keys before truncation
	qb.WriteString("SET SESSION FOREIGN_KEY_CHECKS = 0; ")
	qb.WriteString(strings.Join(stmts, " "))
	// #49: Re-enable foreign keys after truncation
	qb.WriteString(" SET SESSION FOREIGN_KEY_CHECKS = 1;")

	return tx.RawQuery(qb.String()).Exec()
}

func newMySQL(deets *ConnectionDetails) (dialect, error) {
	cd := &mysql{
		commonDialect: commonDialect{ConnectionDetails: deets},
	}
	return cd, nil
}

func urlParserMySQL(cd *ConnectionDetails) error {
	cfg, err := _mysql.ParseDSN(strings.TrimPrefix(cd.URL, "mysql://"))
	if err != nil {
		return fmt.Errorf("the URL '%s' is not supported by MySQL driver: %w", cd.URL, err)
	}

	cd.User = cfg.User
	cd.Password = cfg.Passwd
	cd.Database = cfg.DBName

	// NOTE: use cfg.Params if want to fill options with full parameters
	cd.setOption("collation", cfg.Collation)

	if cfg.Net == "unix" {
		cd.Port = "socket" // trick. see: `URL()`
		cd.Host = cfg.Addr
	} else {
		tmp := strings.Split(cfg.Addr, ":")
		cd.Host = tmp[0]
		if len(tmp) > 1 {
			cd.Port = tmp[1]
		}
	}

	return nil
}

func finalizerMySQL(cd *ConnectionDetails) {
	cd.Host = defaults.String(cd.Host, hostMySQL)
	cd.Port = defaults.String(cd.Port, portMySQL)

	defs := map[string]string{
		"readTimeout": "3s",
		"collation":   "utf8mb4_general_ci",
	}
	forced := map[string]string{
		"parseTime":       "true",
		"multiStatements": "true",
	}

	for k, def := range defs {
		cd.setOptionWithDefault(k, cd.option(k), def)
	}

	for k, v := range forced {
		// respect user specified options but print warning!
		cd.setOptionWithDefault(k, cd.option(k), v)
		if cd.option(k) != v { // when user-defined option exists
			log(logging.Warn, "IMPORTANT! '%s: %s' option is required to work properly but your current setting is '%v: %v'.", k, v, k, cd.option(k))
			log(logging.Warn, "It is highly recommended to remove '%v: %v' option from your config!", k, cd.option(k))
		} // or override with `cd.Options[k] = v`?
		if cd.URL != "" && !strings.Contains(cd.URL, k+"="+v) {
			log(logging.Warn, "IMPORTANT! '%s=%s' option is required to work properly. Please add it to the database URL in the config!", k, v)
		} // or fix user specified url?
	}
}

const mysqlTruncate = "SELECT concat('TRUNCATE TABLE `', TABLE_NAME, '`;') as stmt FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = ? AND table_name <> ? AND table_type <> 'VIEW'"
