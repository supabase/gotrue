package instrumentedsql

import (
	"context"
	"database/sql/driver"
	"time"
)

type wrappedStmt struct {
	opts
	ctx    context.Context
	query  string
	parent driver.Stmt
}

// Compile time validation that our types implement the expected interfaces
var (
	_ driver.Stmt = wrappedStmt{}
	_ driver.StmtExecContext = wrappedStmt{}
	_ driver.StmtQueryContext = wrappedStmt{}
	_ driver.ColumnConverter = wrappedStmt{}
)

func (s wrappedStmt) Close() (err error) {
	if !s.hasOpExcluded(OpSQLStmtClose) {
		span := s.GetSpan(s.ctx).NewChild(OpSQLStmtClose)
		span.SetLabel("component", "database/sql")
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			s.Log(s.ctx, OpSQLStmtClose, "err", err, "duration", time.Since(start))
		}()
	}

	return s.parent.Close()
}

func (s wrappedStmt) NumInput() int {
	return s.parent.NumInput()
}

func (s wrappedStmt) Exec(args []driver.Value) (res driver.Result, err error) {
	if !s.hasOpExcluded(OpSQLStmtExec) {
		span := s.GetSpan(s.ctx).NewChild(OpSQLStmtExec)
		span.SetLabel("component", "database/sql")
		span.SetLabel("query", s.query)
		if !s.OmitArgs {
			span.SetLabel("args", formatArgs(args))
		}
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			logQuery(s.ctx, s.opts, OpSQLStmtExec, s.query, err, args, start)
		}()
	}

	res, err = s.parent.Exec(args)
	if err != nil {
		return nil, err
	}

	return wrappedResult{opts: s.opts, ctx: s.ctx, parent: res}, nil
}

func (s wrappedStmt) Query(args []driver.Value) (rows driver.Rows, err error) {
	if !s.hasOpExcluded(OpSQLStmtQuery) {
		span := s.GetSpan(s.ctx).NewChild(OpSQLStmtQuery)
		span.SetLabel("component", "database/sql")
		span.SetLabel("query", s.query)
		if !s.OmitArgs {
			span.SetLabel("args", formatArgs(args))
		}
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			logQuery(s.ctx, s.opts, OpSQLStmtQuery, s.query, err, args, start)
		}()
	}

	rows, err = s.parent.Query(args)
	if err != nil {
		return nil, err
	}

	return wrappedRows{opts: s.opts, ctx: s.ctx, parent: rows}, nil
}

func (s wrappedStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (res driver.Result, err error) {
	if !s.hasOpExcluded(OpSQLStmtExec) {
		span := s.GetSpan(ctx).NewChild(OpSQLStmtExec)
		span.SetLabel("component", "database/sql")
		span.SetLabel("query", s.query)
		if !s.OmitArgs {
			span.SetLabel("args", formatArgs(args))
		}
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			logQuery(ctx, s.opts, OpSQLStmtExec, s.query, err, args, start)
		}()
	}

	if stmtExecContext, ok := s.parent.(driver.StmtExecContext); ok {
		res, err := stmtExecContext.ExecContext(ctx, args)
		if err != nil {
			return nil, err
		}

		return wrappedResult{opts: s.opts, ctx: ctx, parent: res}, nil
	}

	// Fallback implementation
	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}

	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	res, err = s.parent.Exec(dargs)
	if err != nil {
		return nil, err
	}

	return wrappedResult{opts: s.opts, ctx: ctx, parent: res}, nil
}

func (s wrappedStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (rows driver.Rows, err error) {
	if !s.hasOpExcluded(OpSQLStmtQuery) {
		span := s.GetSpan(ctx).NewChild(OpSQLStmtQuery)
		span.SetLabel("component", "database/sql")
		span.SetLabel("query", s.query)
		if !s.OmitArgs {
			span.SetLabel("args", formatArgs(args))
		}
		start := time.Now()
		defer func() {
			span.SetError(err)
			span.Finish()
			logQuery(ctx, s.opts, OpSQLStmtQuery, s.query, err, args, start)
		}()
	}

	if stmtQueryContext, ok := s.parent.(driver.StmtQueryContext); ok {
		rows, err := stmtQueryContext.QueryContext(ctx, args)
		if err != nil {
			return nil, err
		}

		return wrappedRows{opts: s.opts, ctx: ctx, parent: rows}, nil
	}

	dargs, err := namedValueToValue(args)
	if err != nil {
		return nil, err
	}

	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	rows, err = s.parent.Query(dargs)
	if err != nil {
		return nil, err
	}

	return wrappedRows{opts: s.opts, ctx: ctx, parent: rows}, nil
}

func (s wrappedStmt) ColumnConverter(idx int) driver.ValueConverter {
	if converter, ok := s.parent.(driver.ColumnConverter); ok {
		return converter.ColumnConverter(idx)
	}

	return driver.DefaultParameterConverter
}