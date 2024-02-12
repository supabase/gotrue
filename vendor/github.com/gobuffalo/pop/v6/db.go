package pop

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type dB struct {
	*sqlx.DB
}

func (db *dB) TransactionContext(ctx context.Context) (*Tx, error) {
	return newTX(ctx, db, nil)
}

func (db *dB) Transaction() (*Tx, error) {
	return newTX(context.Background(), db, nil)
}

func (db *dB) TransactionContextOptions(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	return newTX(ctx, db, opts)
}

func (db *dB) Rollback() error {
	return nil
}

func (db *dB) Commit() error {
	return nil
}
