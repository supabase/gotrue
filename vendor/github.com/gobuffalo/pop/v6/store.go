package pop

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Store is an interface that must be implemented in order for Pop
// to be able to use the value as a way of talking to a datastore.
type store interface {
	Select(interface{}, string, ...interface{}) error
	Get(interface{}, string, ...interface{}) error
	NamedExec(string, interface{}) (sql.Result, error)
	Exec(string, ...interface{}) (sql.Result, error)
	PrepareNamed(string) (*sqlx.NamedStmt, error)
	Transaction() (*Tx, error)
	Rollback() error
	Commit() error
	Close() error

	// Context versions to wrap with contextStore
	SelectContext(context.Context, interface{}, string, ...interface{}) error
	GetContext(context.Context, interface{}, string, ...interface{}) error
	NamedExecContext(context.Context, string, interface{}) (sql.Result, error)
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareNamedContext(context.Context, string) (*sqlx.NamedStmt, error)
	TransactionContext(context.Context) (*Tx, error)
	TransactionContextOptions(context.Context, *sql.TxOptions) (*Tx, error)
}

// ContextStore wraps a store with a Context, so passes it with the functions that don't take it.
type contextStore struct {
	store
	ctx context.Context
}

func (s contextStore) Transaction() (*Tx, error) {
	return s.store.TransactionContext(s.ctx)
}
func (s contextStore) Select(dest interface{}, query string, args ...interface{}) error {
	return s.store.SelectContext(s.ctx, dest, query, args...)
}
func (s contextStore) Get(dest interface{}, query string, args ...interface{}) error {
	return s.store.GetContext(s.ctx, dest, query, args...)
}
func (s contextStore) NamedExec(query string, arg interface{}) (sql.Result, error) {
	return s.store.NamedExecContext(s.ctx, query, arg)
}
func (s contextStore) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.store.ExecContext(s.ctx, query, args...)
}
func (s contextStore) PrepareNamed(query string) (*sqlx.NamedStmt, error) {
	return s.store.PrepareNamedContext(s.ctx, query)
}

func (s contextStore) Context() context.Context {
	return s.ctx
}
