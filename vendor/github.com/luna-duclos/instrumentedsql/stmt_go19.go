package instrumentedsql

import "database/sql/driver"

var _ driver.NamedValueChecker = wrappedStmt{}

func (c wrappedStmt) CheckNamedValue(v *driver.NamedValue) error {
	if checker, ok := c.parent.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(v)
	}

	return driver.ErrSkip
}
