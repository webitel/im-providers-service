package store

import (
	"context"
	"time"

	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

type CustomStore interface {
	Insert(ctx context.Context, dc int64, g *custommodel.CustomGate) error
	Select(ctx context.Context, filter custommodel.GateFilter) (*custommodel.CustomGate, error)
	Update(ctx context.Context, req custommodel.UpdateCustom) (*custommodel.CustomGate, error)
	Unbind(ctx context.Context, gateID string) (*custommodel.CustomGate, error)

	UpsertChat(ctx context.Context, gateID, chatID, externalSub string) error
	ChatByRecipient(ctx context.Context, gateID, externalSub string) (*custommodel.Chat, error)
	ChatByID(ctx context.Context, gateID, chatID string) (*custommodel.Chat, error)
}

type OutboxStore interface {
	EnqueueOutbox(ctx context.Context, rec custommodel.OutboxRecord) error
	EnqueueOutboxIfPending(ctx context.Context, rec custommodel.OutboxRecord) (bool, error)
	ClaimOutbox(ctx context.Context, limit int, lease time.Duration) ([]custommodel.OutboxTask, error)
	RetryOutbox(ctx context.Context, id int64, nextAt time.Time, lastErr string) error
	FailOutbox(ctx context.Context, id int64, lastErr string) error
	DeleteOutbox(ctx context.Context, id int64) error
}
