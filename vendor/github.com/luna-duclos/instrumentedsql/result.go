package instrumentedsql

import (
	"context"
	"database/sql/driver"
	"time"
)

type wrappedResult struct {
	opts
	ctx    context.Context
	parent driver.Result
}

func (r wrappedResult) LastInsertId() (id int64, err error) {
	if !r.hasOpExcluded(OpSQLResLastInsertID) {
		span := r.GetSpan(r.ctx).NewChild(OpSQLResLastInsertID)
		span.SetLabel("component", "database/sql")
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			r.Log(r.ctx, OpSQLResLastInsertID, "err", err, "duration", time.Since(start))
		}()
	}

	return r.parent.LastInsertId()
}

func (r wrappedResult) RowsAffected() (num int64, err error) {
	if !r.hasOpExcluded(OpSQLResRowsAffected) {
		span := r.GetSpan(r.ctx).NewChild(OpSQLResRowsAffected)
		span.SetLabel("component", "database/sql")
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			r.Log(r.ctx, OpSQLResRowsAffected, "err", err, "duration", time.Since(start))
		}()
	}

	return r.parent.RowsAffected()
}