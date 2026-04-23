package postgresx

import "github.com/jackc/pgx/v5"

type batchBuilder struct {
	batch pgx.Batch
}

func NewBatchBuilder() *batchBuilder { return &batchBuilder{} }

func (b *batchBuilder) Queue(sql string, args ...any) *pgx.QueuedQuery {
	return b.batch.Queue(sql, args...)
}

func (b *batchBuilder) Len() int          { return b.batch.Len() }
func (b *batchBuilder) Build() *pgx.Batch { return &b.batch }
