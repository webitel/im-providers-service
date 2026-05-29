package postgresx

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type conn struct {
	sourcePool   *pgxpool.Pool
	poolConn     *pgxpool.Conn
	queryTimeout time.Duration
	txTimeout    time.Duration
}

func (c *conn) withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.queryTimeout <= 0 {
		return ctx, func() {}
	}

	if dl, ok := ctx.Deadline(); ok && time.Until(dl) < c.queryTimeout {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.queryTimeout)
}

func (c *conn) withTxTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.txTimeout <= 0 {
		return ctx, func() {}
	}
	if dl, ok := ctx.Deadline(); ok && time.Until(dl) < c.txTimeout {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.txTimeout)
}

func (c *conn) Release() {
	c.poolConn.Release()
}

func (c *conn) Hijack() *pgx.Conn {
	return c.poolConn.Hijack()
}

func (c *conn) Close(ctx context.Context) error {
	raw := c.poolConn.Hijack()
	return raw.Close(ctx)
}

func (c *conn) Ping(ctx context.Context) error {
	ctx, cancel := c.withQueryTimeout(ctx)
	defer cancel()
	return c.poolConn.Ping(ctx)
}

func (c *conn) Conn() *pgx.Conn {
	return c.poolConn.Conn()
}

func (c *conn) PgConn() *pgconn.PgConn {
	return c.poolConn.Conn().PgConn()
}

func (c *conn) Begin(ctx context.Context) (Tx, error) {
	return c.BeginTx(ctx, pgx.TxOptions{})
}

func (c *conn) BeginTx(ctx context.Context, opts pgx.TxOptions) (Tx, error) {
	ctx, cancel := c.withTxTimeout(ctx)
	pgxTx, err := c.poolConn.BeginTx(ctx, opts)
	if err != nil {
		cancel()
		return nil, err
	}
	return newTx(pgxTx, cancel), nil
}

func (c *conn) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := c.withQueryTimeout(ctx)
	defer cancel()
	return c.poolConn.Exec(ctx, sql, args...)
}

func (c *conn) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx, cancel := c.withQueryTimeout(ctx)
	defer cancel()
	return c.poolConn.Query(ctx, sql, args...)
}

func (c *conn) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx, cancel := c.withQueryTimeout(ctx)
	defer cancel()
	return c.poolConn.QueryRow(ctx, sql, args...)
}

func (c *conn) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	ctx, cancel := c.withQueryTimeout(ctx)
	return &batchResultsWithCancel{
		BatchResults: c.poolConn.SendBatch(ctx, b),
		cancel:       cancel,
	}
}

func (c *conn) CopyFrom(
	ctx context.Context,
	table pgx.Identifier,
	cols []string,
	src pgx.CopyFromSource,
) (int64, error) {
	ctx, cancel := c.withQueryTimeout(ctx)
	defer cancel()
	return c.poolConn.CopyFrom(ctx, table, cols, src)
}

func (c *conn) WaitForNotification(ctx context.Context) (*pgconn.Notification, error) {
	return c.poolConn.Conn().WaitForNotification(ctx)
}

func acquireConn(ctx context.Context, pool *pgxpool.Pool, queryTimeout, txTimeout time.Duration) (*conn, error) {
	pc, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &conn{
		sourcePool:   pool,
		poolConn:     pc,
		queryTimeout: queryTimeout,
		txTimeout:    txTimeout,
	}, nil
}
