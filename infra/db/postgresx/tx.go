package postgresx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type transaction struct {
	tx     pgx.Tx
	cancel context.CancelFunc
}

func newTx(t pgx.Tx, cancel context.CancelFunc) *transaction {
	return &transaction{tx: t, cancel: cancel}
}

func (t *transaction) Commit(ctx context.Context) error {
	defer t.cancel()
	return t.tx.Commit(ctx)
}
func (t *transaction) Rollback(ctx context.Context) error {
	defer t.cancel()
	return t.tx.Rollback(ctx)
}
func (t *transaction) Begin(ctx context.Context) (Tx, error) {
	nested, err := t.tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return newTx(nested, func() {}), nil
}

func (t *transaction) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

func (t *transaction) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t *transaction) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *transaction) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return t.tx.SendBatch(ctx, b)
}

func (t *transaction) CopyFrom(
	ctx context.Context,
	table pgx.Identifier,
	cols []string,
	src pgx.CopyFromSource,
) (int64, error) {
	return t.tx.CopyFrom(ctx, table, cols, src)
}

func (t *transaction) Conn() *pgx.Conn {
	return t.tx.Conn()
}
