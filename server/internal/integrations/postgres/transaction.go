package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// WithTx executes fn in a PostgreSQL transaction.
//
// If fn returns an error, the transaction is rolled back. If fn succeeds,
// the transaction is committed. The transaction is always rolled back on
// return as a safety net for every path that does not commit successfully.
func (c *Client) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	if c == nil || c.pool == nil {
		return fmt.Errorf("postgres client is nil")
	}

	if fn == nil {
		return fmt.Errorf("transaction function is nil")
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin postgres transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return fmt.Errorf("execute postgres transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit postgres transaction: %w", err)
	}

	return nil
}
