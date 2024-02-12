package instrumentedsql

import "database/sql/driver"

// WrappedDriver wraps a driver and adds instrumentation.
// Use WrapDriver to create a new WrappedDriver.
type WrappedDriver struct {
	opts
	parent driver.Driver
}

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Driver = WrappedDriver{}
)

// WrapDriver will wrap the passed SQL driver and return a new sql driver that uses it and also logs and traces calls using the passed logger and tracer
// The returned driver will still have to be registered with the sql package before it can be used.
//
// Important note: Seeing as the context passed into the various instrumentation calls this package calls,
// Any call without a context passed will not be instrumented. Please be sure to use the ___Context() and BeginTx() function calls added in Go 1.8
// instead of the older calls which do not accept a context.
func WrapDriver(driver driver.Driver, opts ...Opt) WrappedDriver {
	d := WrappedDriver{parent: driver}

	for _, opt := range opts {
		opt(&d.opts)
	}

	if d.Logger == nil {
		d.Logger = nullLogger{}
	}
	if d.Tracer == nil {
		d.Tracer = nullTracer{}
	}

	return d
}

// Open implements the database/sql/driver.Driver interface for WrappedDriver.
func (d WrappedDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.parent.Open(name)
	if err != nil {
		return nil, err
	}

	return wrappedConn{opts: d.opts, parent: conn}, nil
}
