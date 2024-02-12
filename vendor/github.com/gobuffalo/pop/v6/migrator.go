package pop

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/gobuffalo/pop/v6/logging"
)

var mrx = regexp.MustCompile(`^(\d+)_([^.]+)(\.[a-z0-9]+)?\.(up|down)\.(sql|fizz)$`)

// NewMigrator returns a new "blank" migrator. It is recommended
// to use something like MigrationBox or FileMigrator. A "blank"
// Migrator should only be used as the basis for a new type of
// migration system.
func NewMigrator(c *Connection) Migrator {
	return Migrator{
		Connection: c,
	}
}

// Migrator forms the basis of all migrations systems.
// It does the actual heavy lifting of running migrations.
// When building a new migration system, you should embed this
// type into your migrator.
type Migrator struct {
	Connection     *Connection
	SchemaPath     string
	UpMigrations   UpMigrations
	DownMigrations DownMigrations
}

func (m Migrator) migrationIsCompatible(d dialect, mi Migration) bool {
	if mi.DBType == "all" || mi.DBType == d.Name() {
		return true
	}
	return false
}

// UpLogOnly insert pending "up" migrations logs only, without applying the patch.
// It's used when loading the schema dump, instead of the migrations.
func (m Migrator) UpLogOnly() error {
	c := m.Connection
	return m.exec(func() error {
		mtn := c.MigrationTableName()
		mfs := m.UpMigrations
		sort.Sort(mfs)
		return c.Transaction(func(tx *Connection) error {
			for _, mi := range mfs.Migrations {
				if !m.migrationIsCompatible(c.Dialect, mi) {
					continue
				}
				exists, err := c.Where("version = ?", mi.Version).Exists(mtn)
				if err != nil {
					return fmt.Errorf("problem checking for migration version %s: %w", mi.Version, err)
				}
				if exists {
					continue
				}
				_, err = tx.Store.Exec(fmt.Sprintf("insert into %s (version) values ('%s')", mtn, mi.Version))
				if err != nil {
					return fmt.Errorf("problem inserting migration version %s: %w", mi.Version, err)
				}
			}
			return nil
		})
	})
}

// Up runs pending "up" migrations and applies them to the database.
func (m Migrator) Up() error {
	_, err := m.UpTo(0)
	return err
}

// UpTo runs up to step "up" migrations and applies them to the database.
// If step <= 0 all pending migrations are run.
func (m Migrator) UpTo(step int) (applied int, err error) {
	c := m.Connection
	err = m.exec(func() error {
		mtn := c.MigrationTableName()
		mfs := m.UpMigrations
		mfs.Filter(func(mf Migration) bool {
			return m.migrationIsCompatible(c.Dialect, mf)
		})
		sort.Sort(mfs)
		for _, mi := range mfs.Migrations {
			exists, err := c.Where("version = ?", mi.Version).Exists(mtn)
			if err != nil {
				return fmt.Errorf("problem checking for migration version %s: %w", mi.Version, err)
			}
			if exists {
				continue
			}
			runStep := func(tx *Connection) error {
				err := mi.Run(tx)
				if err != nil {
					return err
				}
				_, err = tx.Store.Exec(fmt.Sprintf("insert into %s (version) values ('%s')", mtn, mi.Version))
				if err != nil {
					return fmt.Errorf("problem inserting migration version %s: %w", mi.Version, err)
				}
				return nil
			}
			if c.TX != nil {
				// connection is already in a transaction, no
				// need for nested transactions which don't
				// work well with most dialects
				err = runStep(c)
			} else {
				err = c.Transaction(runStep)
			}
			if err != nil {
				return err
			}
			log(logging.Info, "> %s", mi.Name)
			applied++
			if step > 0 && applied >= step {
				break
			}
		}
		if applied == 0 {
			log(logging.Info, "Migrations already up to date, nothing to apply")
		} else {
			log(logging.Info, "Successfully applied %d migrations.", applied)
		}
		return nil
	})
	return
}

// Down runs pending "down" migrations and rolls back the
// database by the specified number of steps.
func (m Migrator) Down(step int) error {
	c := m.Connection
	return m.exec(func() error {
		mtn := c.MigrationTableName()
		count, err := c.Count(mtn)
		if err != nil {
			return fmt.Errorf("migration down: unable count existing migration: %w", err)
		}
		mfs := m.DownMigrations
		mfs.Filter(func(mf Migration) bool {
			return m.migrationIsCompatible(c.Dialect, mf)
		})
		sort.Sort(mfs)
		// skip all ran migration
		if len(mfs.Migrations) > count {
			mfs.Migrations = mfs.Migrations[len(mfs.Migrations)-count:]
		}
		// run only required steps
		if step > 0 && len(mfs.Migrations) >= step {
			mfs.Migrations = mfs.Migrations[:step]
		}
		for _, mi := range mfs.Migrations {
			exists, err := c.Where("version = ?", mi.Version).Exists(mtn)
			if err != nil {
				return fmt.Errorf("problem checking for migration version %s: %w", mi.Version, err)
			}
			if !exists {
				return fmt.Errorf("migration version %s does not exist", mi.Version)
			}
			runStep := func(tx *Connection) error {
				err := mi.Run(tx)
				if err != nil {
					return err
				}
				err = tx.RawQuery(fmt.Sprintf("delete from %s where version = ?", mtn), mi.Version).Exec()
				if err != nil {
					return fmt.Errorf("problem deleting migration version %s: %w", mi.Version, err)
				}
				return nil
			}
			if c.TX != nil {
				// connection is already in a transaction, no
				// need for nested transactions which don't
				// work well with most dialects
				err = runStep(c)
			} else {
				err = c.Transaction(runStep)
			}
			if err != nil {
				return err
			}

			log(logging.Info, "< %s", mi.Name)
		}
		return nil
	})
}

// Reset the database by running the down migrations followed by the up migrations.
func (m Migrator) Reset() error {
	err := m.Down(-1)
	if err != nil {
		return err
	}
	return m.Up()
}

// CreateSchemaMigrations sets up a table to track migrations. This is an idempotent
// operation.
func CreateSchemaMigrations(c *Connection) error {
	mtn := c.MigrationTableName()
	err := c.Open()
	if err != nil {
		return fmt.Errorf("could not open connection: %w", err)
	}
	_, err = c.Store.Exec(fmt.Sprintf("select * from %s", mtn))
	if err == nil {
		return nil
	}

	return c.Transaction(func(tx *Connection) error {
		schemaMigrations := newSchemaMigrations(mtn)
		smSQL, err := c.Dialect.FizzTranslator().CreateTable(schemaMigrations)
		if err != nil {
			return fmt.Errorf("could not build SQL for schema migration table: %w", err)
		}
		err = tx.RawQuery(smSQL).Exec()
		if err != nil {
			return fmt.Errorf("could not execute %s: %w", smSQL, err)
		}
		return nil
	})
}

// CreateSchemaMigrations sets up a table to track migrations. This is an idempotent
// operation.
func (m Migrator) CreateSchemaMigrations() error {
	return CreateSchemaMigrations(m.Connection)
}

// Status prints out the status of applied/pending migrations.
func (m Migrator) Status(out io.Writer) error {
	err := m.CreateSchemaMigrations()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', tabwriter.TabIndent)
	_, _ = fmt.Fprintln(w, "Version\tName\tStatus\t")
	for _, mf := range m.UpMigrations.Migrations {
		exists, err := m.Connection.Where("version = ?", mf.Version).Exists(m.Connection.MigrationTableName())
		if err != nil {
			return fmt.Errorf("problem with migration: %w", err)
		}
		state := "Pending"
		if exists {
			state = "Applied"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t\n", mf.Version, mf.Name, state)
	}
	return w.Flush()
}

// DumpMigrationSchema will generate a file of the current database schema
// based on the value of Migrator.SchemaPath
func (m Migrator) DumpMigrationSchema() error {
	if m.SchemaPath == "" {
		return nil
	}
	c := m.Connection
	schema := filepath.Join(m.SchemaPath, "schema.sql")
	f, err := os.Create(schema)
	if err != nil {
		return err
	}
	err = c.Dialect.DumpSchema(f)
	if err != nil {
		os.RemoveAll(schema)
		return err
	}
	return nil
}

func (m Migrator) exec(fn func() error) error {
	now := time.Now()
	defer func() {
		err := m.DumpMigrationSchema()
		if err != nil {
			log(logging.Warn, "Migrator: unable to dump schema: %v", err)
		}
	}()
	defer printTimer(now)

	err := m.CreateSchemaMigrations()
	if err != nil {
		return fmt.Errorf("Migrator: problem creating schema migrations: %w", err)
	}
	return fn()
}

func printTimer(timerStart time.Time) {
	diff := time.Since(timerStart).Seconds()
	if diff > 60 {
		log(logging.Info, "%.4f minutes", diff/60)
	} else {
		log(logging.Info, "%.4f seconds", diff)
	}
}
