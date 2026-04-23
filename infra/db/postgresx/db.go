package postgresx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	Close()
	Ping(ctx context.Context) error
	Config() *pgxpool.Config
	Stat() *pgxpool.Stat

	Acquire(ctx context.Context) (Conn, error)
	AcquireFunc(ctx context.Context, f func(Conn) error) error
	Begin(ctx context.Context) (Tx, error)
	BeginTx(ctx context.Context, opts pgx.TxOptions) (Tx, error)

	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults

	CopyFrom(ctx context.Context, table pgx.Identifier, cols []string, src pgx.CopyFromSource) (int64, error)

	PrimaryPool() *pgxpool.Pool
	ReplicaPools() []*pgxpool.Pool
	Replica() Executor
	Primary() Executor
}

type Conn interface {
	Release()
	Hijack() *pgx.Conn
	Ping(ctx context.Context) error
	Conn() *pgx.Conn

	Begin(ctx context.Context) (Tx, error)
	BeginTx(ctx context.Context, opts pgx.TxOptions) (Tx, error)

	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, table pgx.Identifier, cols []string, src pgx.CopyFromSource) (int64, error)

	WaitForNotification(ctx context.Context) (*pgconn.Notification, error)

	PgConn() *pgconn.PgConn
}

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	Begin(ctx context.Context) (Tx, error)

	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(ctx context.Context, table pgx.Identifier, cols []string, src pgx.CopyFromSource) (int64, error)

	Conn() *pgx.Conn
}

type Executor interface {
	CopyFrom(
		ctx context.Context,
		table pgx.Identifier,
		cols []string,
		src pgx.CopyFromSource,
	) (int64, error)
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type RowScanner[T any] interface {
	ScanRow(rows pgx.Rows) (T, error)
}

type BatchBuilder interface {
	Queue(sql string, args ...any) *pgx.QueuedQuery
	Len() int
	Build() *pgx.Batch
}

type CopyFromSource = pgx.CopyFromSource

type TxOptions = pgx.TxOptions

type (
	TxIsoLevel       = pgx.TxIsoLevel
	TxAccessMode     = pgx.TxAccessMode
	TxDeferrableMode = pgx.TxDeferrableMode
)

const (
	Serializable    TxIsoLevel = pgx.Serializable
	RepeatableRead  TxIsoLevel = pgx.RepeatableRead
	ReadCommitted   TxIsoLevel = pgx.ReadCommitted
	ReadUncommitted TxIsoLevel = pgx.ReadUncommitted

	ReadWrite TxAccessMode = pgx.ReadWrite
	ReadOnly  TxAccessMode = pgx.ReadOnly

	Deferrable    TxDeferrableMode = pgx.Deferrable
	NotDeferrable TxDeferrableMode = pgx.NotDeferrable
)

var (
	TxReadWrite    = pgx.TxOptions{IsoLevel: ReadCommitted, AccessMode: ReadWrite}
	TxReadOnly     = pgx.TxOptions{IsoLevel: RepeatableRead, AccessMode: ReadOnly}
	TxSerializable = pgx.TxOptions{IsoLevel: Serializable, AccessMode: ReadWrite}
)

type NamedArgs = pgx.NamedArgs
