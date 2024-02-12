package pop

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"

	mysqld "github.com/go-sql-driver/mysql"
	"github.com/gobuffalo/pop/v6/logging"
	pgx "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/luna-duclos/instrumentedsql"
)

const instrumentedDriverName = "instrumented-sql-driver"

var sqlDriverLock = sync.Mutex{}

func instrumentDriver(deets *ConnectionDetails, defaultDriverName string) (driverName, dialect string, err error) {
	driverName = defaultDriverName
	if deets.Driver != "" {
		driverName = deets.Driver
	}
	dialect = driverName

	if !deets.UseInstrumentedDriver {
		if len(deets.InstrumentedDriverOptions) > 0 {
			log(logging.Warn, "SQL driver instrumentation is disabled but `ConnectionDetails.InstrumentedDriverOptions` is not empty. Please double-check if this is a error.")
		}

		// If instrumentation is disabled, we just return the driver name we got (e.g. "pgx").
		return driverName, dialect, nil
	}

	if len(deets.InstrumentedDriverOptions) == 0 {
		log(logging.Warn, "SQL driver instrumentation was enabled but no options have been passed to `ConnectionDetails.InstrumentedDriverOptions`. Instrumentation will therefore not result in any output.")
	}

	var dr driver.Driver
	var newDriverName string
	switch CanonicalDialect(driverName) {
	case nameCockroach:
		fallthrough
	case namePostgreSQL:
		dr = new(pgx.Driver)
		newDriverName = instrumentedDriverName + "-" + namePostgreSQL
	case nameMariaDB:
		fallthrough
	case nameMySQL:
		dr = mysqld.MySQLDriver{}
		newDriverName = instrumentedDriverName + "-" + nameMySQL
	case nameSQLite3:
		var err error
		dr, err = newSQLiteDriver()
		if err != nil {
			return "", "", err
		}
		newDriverName = instrumentedDriverName + "-" + nameSQLite3
	}

	sqlDriverLock.Lock()
	defer sqlDriverLock.Unlock()

	var found bool
	for _, n := range sql.Drivers() {
		if n == newDriverName {
			found = true
			break
		}
	}

	if !found {
		sql.Register(newDriverName, instrumentedsql.WrapDriver(dr, deets.InstrumentedDriverOptions...))
	}

	return newDriverName, dialect, nil
}

// openPotentiallyInstrumentedConnection first opens a raw SQL connection and then wraps it with `sqlx`.
//
// We do this because `sqlx` needs the database type in order to properly
// translate arguments (e.g. `?` to `$1`) in SQL queries. Because we use
// a custom driver name when using instrumentation, this detection would fail
// otherwise.
func openPotentiallyInstrumentedConnection(c dialect, dsn string) (*sqlx.DB, error) {
	driverName, dialect, err := instrumentDriver(c.Details(), c.DefaultDriver())
	if err != nil {
		return nil, err
	}

	con, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("could not open database connection: %w", err)
	}

	return sqlx.NewDb(con, dialect), nil
}
