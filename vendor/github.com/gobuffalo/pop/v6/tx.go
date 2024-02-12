package pop

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Tx stores a transaction with an ID to keep track.
type Tx struct {
	ID int
	*sqlx.Tx
}

func newTX(ctx context.Context, db *dB, opts *sql.TxOptions) (*Tx, error) {
	t := &Tx{
		ID: rand.Int(),
	}
	tx, err := db.BeginTxx(ctx, opts)
	t.Tx = tx
	if err != nil {
		return nil, fmt.Errorf("could not create new transaction: %w", err)
	}
	return t, nil
}

// TransactionContext simply returns the current transaction,
// this is defined so it implements the `Store` interface.
func (tx *Tx) TransactionContext(ctx context.Context) (*Tx, error) {
	return tx, nil
}

// TransactionContextOptions simply returns the current transaction,
// this is defined so it implements the `Store` interface.
func (tx *Tx) TransactionContextOptions(_ context.Context, _ *sql.TxOptions) (*Tx, error) {
	return tx, nil
}

// Transaction simply returns the current transaction,
// this is defined so it implements the `Store` interface.
func (tx *Tx) Transaction() (*Tx, error) {
	return tx, nil
}

// Close does nothing. This is defined so it implements the `Store` interface.
func (tx *Tx) Close() error {
	return nil
}
