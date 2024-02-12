// Copyright Sam Xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelsql

import (
	"context"
	"database/sql/driver"

	"go.opentelemetry.io/otel/trace"
)

var (
	_ driver.Stmt              = (*otStmt)(nil)
	_ driver.StmtExecContext   = (*otStmt)(nil)
	_ driver.StmtQueryContext  = (*otStmt)(nil)
	_ driver.NamedValueChecker = (*otStmt)(nil)
)

type otStmt struct {
	driver.Stmt
	cfg config

	query string
}

func newStmt(stmt driver.Stmt, cfg config, query string) *otStmt {
	return &otStmt{
		Stmt:  stmt,
		cfg:   cfg,
		query: query,
	}
}

func (s *otStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (result driver.Result, err error) {
	execer, ok := s.Stmt.(driver.StmtExecContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	method := MethodStmtExec
	onDefer := recordMetric(ctx, s.cfg.Instruments, s.cfg.Attributes, method)
	defer func() {
		onDefer(err)
	}()

	var span trace.Span
	ctx, span = s.cfg.Tracer.Start(ctx, s.cfg.SpanNameFormatter.Format(ctx, method, s.query),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(withDBStatement(s.cfg, s.query)...),
	)
	defer span.End()

	result, err = execer.ExecContext(ctx, args)
	if err != nil {
		recordSpanError(span, s.cfg.SpanOptions, err)
		return nil, err
	}
	return result, nil
}

func (s *otStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (rows driver.Rows, err error) {
	query, ok := s.Stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	method := MethodStmtQuery
	onDefer := recordMetric(ctx, s.cfg.Instruments, s.cfg.Attributes, method)
	defer func() {
		onDefer(err)
	}()

	queryCtx, span := s.cfg.Tracer.Start(ctx, s.cfg.SpanNameFormatter.Format(ctx, method, s.query),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(withDBStatement(s.cfg, s.query)...),
	)
	defer span.End()

	rows, err = query.QueryContext(queryCtx, args)
	if err != nil {
		recordSpanError(span, s.cfg.SpanOptions, err)
		return nil, err
	}
	return newRows(ctx, rows, s.cfg), nil
}

func (s *otStmt) CheckNamedValue(namedValue *driver.NamedValue) error {
	namedValueChecker, ok := s.Stmt.(driver.NamedValueChecker)
	if !ok {
		return driver.ErrSkip
	}

	return namedValueChecker.CheckNamedValue(namedValue)
}
