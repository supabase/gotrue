// +build go1.10

package instrumentedsql

import (
	"context"
	"database/sql/driver"
	"time"
)

type wrappedConnector struct {
	opts
	parent    driver.Connector
	driverRef *WrappedDriver
}

var (
	_ driver.Connector = wrappedConnector{}
)

func (c wrappedConnector) Connect(ctx context.Context) (conn driver.Conn, err error) {
	if !c.hasOpExcluded(OpSQLConnectorConnect) {
		span := c.GetSpan(ctx).NewChild(OpSQLConnectorConnect)
		span.SetLabel("component", "database/sql")
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			c.Log(ctx, OpSQLConnectorConnect, "err", err, "duration", time.Since(start))
		}()
	}

	conn, err = c.parent.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return wrappedConn{opts: c.driverRef.opts, parent: conn}, nil
}

func (c wrappedConnector) Driver() driver.Driver {
	return c.driverRef
}

// dsnConnector is a fallback connector placed in position of wrappedConnector.parent
// when given Driver does not comply with DriverContext interface.
type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

func (t dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return t.driver.Open(t.dsn)
}

func (t dsnConnector) Driver() driver.Driver {
	return t.driver
}
