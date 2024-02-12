package instrumentedsql

import (
	"context"
	"database/sql/driver"
	"io"
	"time"
)

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Rows                           = wrappedRows{}
	_ driver.RowsColumnTypeDatabaseTypeName // TODO
	_ driver.RowsColumnTypeLength           // TODO
	_ driver.RowsColumnTypeNullable         // TODO
	_ driver.RowsColumnTypePrecisionScale   // TODO
	_ driver.RowsColumnTypeScanType         // TODO
	_ driver.RowsNextResultSet              // TODO
)

type wrappedRows struct {
	opts
	ctx    context.Context
	parent driver.Rows
}

func (r wrappedRows) Columns() []string {
	return r.parent.Columns()
}

func (r wrappedRows) Close() error {
	return r.parent.Close()
}

func (r wrappedRows) Next(dest []driver.Value) (err error) {
	if !r.hasOpExcluded(OpSQLRowsNext) {
		span := r.GetSpan(r.ctx).NewChild(OpSQLRowsNext)
		span.SetLabel("component", "database/sql")
		defer func() {
			if err != io.EOF {
				span.SetError(err)
			}
			span.Finish()
		}()

		start := time.Now()
		defer func() {
			r.Log(r.ctx, OpSQLRowsNext, "err", err, "duration", time.Since(start))
		}()
	}

	return r.parent.Next(dest)
}
