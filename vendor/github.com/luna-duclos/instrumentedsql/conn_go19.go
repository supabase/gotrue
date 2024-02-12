// +build go1.9

package instrumentedsql

import "database/sql/driver"

var (
	_ driver.NamedValueChecker = wrappedConn{}
)

func (c wrappedConn) CheckNamedValue(v *driver.NamedValue) error {
	if checker, ok := c.parent.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(v)
	}

	return driver.ErrSkip
}