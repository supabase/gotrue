package pop

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/gobuffalo/pop/v6/columns"
	"github.com/gobuffalo/pop/v6/logging"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

func init() {
	gob.Register(uuid.UUID{})
}

type commonDialect struct {
	ConnectionDetails *ConnectionDetails
}

func (commonDialect) Lock(fn func() error) error {
	return fn()
}

func (commonDialect) Quote(key string) string {
	parts := strings.Split(key, ".")

	for i, part := range parts {
		part = strings.Trim(part, `"`)
		part = strings.TrimSpace(part)

		parts[i] = fmt.Sprintf(`"%v"`, part)
	}

	return strings.Join(parts, ".")
}

func genericCreate(c *Connection, model *Model, cols columns.Columns, quoter quotable) error {
	keyType, err := model.PrimaryKeyType()
	if err != nil {
		return err
	}
	switch keyType {
	case "int", "int64":
		var id int64
		cols.Remove(model.IDField())
		w := cols.Writeable()
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoter.Quote(model.TableName()), w.QuotedString(quoter), w.SymbolizedString())
		txlog(logging.SQL, c, query, model.Value)
		res, err := c.Store.NamedExec(query, model.Value)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		if err == nil {
			model.setID(id)
		}
		if err != nil {
			return err
		}
		return nil
	case "UUID", "string":
		if keyType == "UUID" {
			if model.ID() == emptyUUID {
				u, err := uuid.NewV4()
				if err != nil {
					return err
				}
				model.setID(u)
			}
		} else if model.ID() == "" {
			return fmt.Errorf("missing ID value")
		}
		w := cols.Writeable()
		w.Add(model.IDField())
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoter.Quote(model.TableName()), w.QuotedString(quoter), w.SymbolizedString())
		txlog(logging.SQL, c, query, model.Value)
		stmt, err := c.Store.PrepareNamed(query)
		if err != nil {
			return err
		}
		_, err = stmt.ExecContext(model.ctx, model.Value)
		if err != nil {
			if closeErr := stmt.Close(); closeErr != nil {
				return fmt.Errorf("failed to close prepared statement: %s: %w", closeErr, err)
			}
			return err
		}
		if err := stmt.Close(); err != nil {
			return fmt.Errorf("failed to close statement: %w", err)
		}
		return nil
	}
	return fmt.Errorf("can not use %s as a primary key type!", keyType)
}

func genericUpdate(c *Connection, model *Model, cols columns.Columns, quoter quotable) error {
	stmt := fmt.Sprintf("UPDATE %s AS %s SET %s WHERE %s", quoter.Quote(model.TableName()), model.Alias(), cols.Writeable().QuotedUpdateString(quoter), model.WhereNamedID())
	txlog(logging.SQL, c, stmt, model.ID())
	_, err := c.Store.NamedExec(stmt, model.Value)
	if err != nil {
		return err
	}
	return nil
}

func genericUpdateQuery(c *Connection, model *Model, cols columns.Columns, quoter quotable, query Query, bindType int) (int64, error) {
	q := fmt.Sprintf("UPDATE %s AS %s SET %s", quoter.Quote(model.TableName()), model.Alias(), cols.Writeable().QuotedUpdateString(quoter))

	q, updateArgs, err := sqlx.Named(q, model.Value)
	if err != nil {
		return 0, err
	}

	sb := query.toSQLBuilder(model)
	q = sb.buildWhereClauses(q)

	q = sqlx.Rebind(bindType, q)

	result, err := genericExec(c, q, append(updateArgs, sb.args...)...)
	if err != nil {
		return 0, err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return n, err
}

func genericDestroy(c *Connection, model *Model, quoter quotable) error {
	stmt := fmt.Sprintf("DELETE FROM %s AS %s WHERE %s", quoter.Quote(model.TableName()), model.Alias(), model.WhereID())
	_, err := genericExec(c, stmt, model.ID())
	if err != nil {
		return err
	}
	return nil
}

func genericDelete(c *Connection, model *Model, query Query) error {
	sqlQuery, args := query.ToSQL(model)
	_, err := genericExec(c, sqlQuery, args...)
	return err
}

func genericExec(c *Connection, stmt string, args ...interface{}) (sql.Result, error) {
	txlog(logging.SQL, c, stmt, args...)
	res, err := c.Store.Exec(stmt, args...)
	return res, err
}

func genericSelectOne(c *Connection, model *Model, query Query) error {
	sqlQuery, args := query.ToSQL(model)
	txlog(logging.SQL, query.Connection, sqlQuery, args...)
	err := c.Store.Get(model.Value, sqlQuery, args...)
	if err != nil {
		return err
	}
	return nil
}

func genericSelectMany(c *Connection, models *Model, query Query) error {
	sqlQuery, args := query.ToSQL(models)
	txlog(logging.SQL, query.Connection, sqlQuery, args...)
	err := c.Store.Select(models.Value, sqlQuery, args...)
	if err != nil {
		return err
	}
	return nil
}

func genericLoadSchema(d dialect, r io.Reader) error {
	deets := d.Details()

	// Open DB connection on the target DB
	db, err := openPotentiallyInstrumentedConnection(d, d.MigrationURL())
	if err != nil {
		return fmt.Errorf("unable to load schema for %s: %w", deets.Database, err)
	}
	defer db.Close()

	// Get reader contents
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if len(contents) == 0 {
		log(logging.Info, "schema is empty for %s, skipping", deets.Database)
		return nil
	}

	_, err = db.Exec(string(contents))
	if err != nil {
		return fmt.Errorf("unable to load schema for %s: %w", deets.Database, err)
	}

	log(logging.Info, "loaded schema for %s", deets.Database)
	return nil
}

func genericDumpSchema(deets *ConnectionDetails, cmd *exec.Cmd, w io.Writer) error {
	log(logging.SQL, strings.Join(cmd.Args, " "))

	bb := &bytes.Buffer{}
	mw := io.MultiWriter(w, bb)

	cmd.Stdout = mw
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}

	x := bytes.TrimSpace(bb.Bytes())
	if len(x) == 0 {
		return fmt.Errorf("unable to dump schema for %s", deets.Database)
	}

	log(logging.Info, "dumped schema for %s", deets.Database)
	return nil
}
