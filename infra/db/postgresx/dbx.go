package postgresx

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type database struct {
	primary      *pgxpool.Pool
	replicas     []*pgxpool.Pool
	lb           LoadBalancer
	queryTimeout time.Duration
	txTimeout    time.Duration
}

var _ DB = (*database)(nil)

func (d *database) withQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if d.queryTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d.queryTimeout)
}

func (d *database) withTxTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if d.txTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d.txTimeout)
}

func (d *database) readPool() *pgxpool.Pool {
	if len(d.replicas) == 0 {
		return d.primary
	}
	return d.lb.Resolve(d.replicas)
}

func (d *database) Primary() Executor {
	return &poolExecutor{pool: d.primary, db: d}
}

func (d *database) Replica() Executor {
	return &poolExecutor{pool: d.readPool(), db: d, fallback: d.primary}
}

func (d *database) Close() {
	d.primary.Close()
	for _, r := range d.replicas {
		r.Close()
	}
}

func (d *database) Ping(ctx context.Context) error {
	ctx, cancel := d.withQueryTimeout(ctx)
	defer cancel()
	return d.primary.Ping(ctx)
}

func (d *database) Config() *pgxpool.Config       { return d.primary.Config() }
func (d *database) Stat() *pgxpool.Stat           { return d.primary.Stat() }
func (d *database) PrimaryPool() *pgxpool.Pool    { return d.primary }
func (d *database) ReplicaPools() []*pgxpool.Pool { return d.replicas }

func (d *database) Acquire(ctx context.Context) (Conn, error) {
	return acquireConn(ctx, d.primary, d.queryTimeout, d.txTimeout)
}

func (d *database) AcquireFunc(ctx context.Context, f func(Conn) error) error {
	c, err := d.Acquire(ctx)
	if err != nil {
		return err
	}
	defer c.Release()
	return f(c)
}

func (d *database) Begin(ctx context.Context) (Tx, error) {
	return d.BeginTx(ctx, pgx.TxOptions{})
}

func (d *database) BeginTx(ctx context.Context, opts pgx.TxOptions) (Tx, error) {
	ctx, cancel := d.withTxTimeout(ctx)
	t, err := d.primary.BeginTx(ctx, opts)
	if err != nil {
		cancel()
		return nil, err
	}
	return newTx(t, cancel), nil
}

func (d *database) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return d.Primary().Exec(ctx, sql, args...)
}

func (d *database) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return d.Primary().Query(ctx, sql, args...)
}

func (d *database) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return d.Primary().QueryRow(ctx, sql, args...)
}

func (d *database) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return d.Primary().SendBatch(ctx, b)
}

func (d *database) CopyFrom(
	ctx context.Context,
	table pgx.Identifier,
	cols []string,
	src pgx.CopyFromSource,
) (int64, error) {
	return d.Primary().CopyFrom(ctx, table, cols, src)
}

type poolExecutor struct {
	pool     *pgxpool.Pool
	fallback *pgxpool.Pool
	db       *database
}

func (e *poolExecutor) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := e.db.withQueryTimeout(ctx)
	defer cancel()
	return e.pool.Exec(ctx, sql, args...)
}

func (e *poolExecutor) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	ctx, cancel := e.db.withQueryTimeout(ctx)

	rows, err := e.pool.Query(ctx, sql, args...)
	if err != nil {
		cancel()
		return nil, err
	}

	if isConnError(err) && e.fallback != nil && e.fallback != e.pool {
		cancel()

		ctx, cancel = e.db.withQueryTimeout(ctx)
		rows, err = e.fallback.Query(ctx, sql, args...)
		if err != nil {
			cancel()
			return nil, err
		}
	}

	return &rowsWithCancel{
		Rows:   rows,
		cancel: cancel,
	}, nil
}

func (e *poolExecutor) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	ctx, cancel := e.db.withQueryTimeout(ctx)

	row := e.pool.QueryRow(ctx, sql, args...)

	return &rowWithCancel{
		Row:    row,
		cancel: cancel,
	}
}

func (e *poolExecutor) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	ctx, cancel := e.db.withQueryTimeout(ctx)
	return &batchResultsWithCancel{
		BatchResults: e.pool.SendBatch(ctx, b),
		cancel:       cancel,
	}
}

func (e *poolExecutor) CopyFrom(
	ctx context.Context,
	table pgx.Identifier,
	cols []string,
	src pgx.CopyFromSource,
) (int64, error) {
	ctx, cancel := e.db.withQueryTimeout(ctx)
	defer cancel()
	return e.pool.CopyFrom(ctx, table, cols, src)
}

func isConnError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, "08") || pgErr.Code == "57P01"
	}
	return false
}

type batchResultsWithCancel struct {
	pgx.BatchResults
	cancel context.CancelFunc
}

func (b *batchResultsWithCancel) Close() error {
	defer b.cancel()
	return b.BatchResults.Close()
}

type rowsWithCancel struct {
	pgx.Rows
	cancel context.CancelFunc
}

func (r *rowsWithCancel) Close() {
	defer r.cancel()
	r.Rows.Close()
}

type rowWithCancel struct {
	pgx.Row
	cancel context.CancelFunc
}

func (r *rowWithCancel) Scan(dest ...any) error {
	defer r.cancel()
	return r.Row.Scan(dest...)
}
