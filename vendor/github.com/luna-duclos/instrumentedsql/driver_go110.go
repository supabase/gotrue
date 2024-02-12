// +build go1.10

package instrumentedsql

import "database/sql/driver"

var _ driver.DriverContext = WrappedDriver{}

func (d WrappedDriver) OpenConnector(name string) (driver.Connector, error) {
	driver, ok := d.parent.(driver.DriverContext)
	if !ok {
		return wrappedConnector{
			opts:      d.opts,
			parent:    dsnConnector{dsn: name, driver: d.parent},
			driverRef: &d,
		}, nil
	}
	conn, err := driver.OpenConnector(name)
	if err != nil {
		return nil, err
	}

	return wrappedConnector{opts: d.opts, parent: conn, driverRef: &d}, nil
}
