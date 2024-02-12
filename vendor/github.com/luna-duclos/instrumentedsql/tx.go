package instrumentedsql

import (
	"context"
	"database/sql/driver"
	"time"
)

type wrappedTx struct {
	opts
	ctx    context.Context
	parent driver.Tx
}

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Tx = wrappedTx{}
)

func (t wrappedTx) Commit() (err error) {
	if !t.hasOpExcluded(OpSQLTxCommit) {
		span := t.GetSpan(t.ctx).NewChild(OpSQLTxCommit)
		span.SetLabel("component", "database/sql")
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			t.Log(t.ctx, OpSQLTxCommit, "err", err, "duration", time.Since(start))
		}()
	}

	return t.parent.Commit()
}

func (t wrappedTx) Rollback() (err error) {
	if !t.hasOpExcluded(OpSQLTxRollback) {
		span := t.GetSpan(t.ctx).NewChild(OpSQLTxRollback)
		span.SetLabel("component", "database/sql")
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			t.Log(t.ctx, OpSQLTxRollback, "err", err, "duration", time.Since(start))
		}()
	}

	return t.parent.Rollback()
}