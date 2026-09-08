package store

import (
	"context"

	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

type CustomStore interface {
	Insert(ctx context.Context, dc int64, g *custommodel.CustomGate) error
	Select(ctx context.Context, id string) (*custommodel.CustomGate, error)
	SelectByURI(ctx context.Context, uri string) (*custommodel.CustomGate, error)
	Update(ctx context.Context, g *custommodel.CustomGate) error
	Unbind(ctx context.Context, gateID string) error

	UpsertChat(ctx context.Context, gateID, chatID, externalSub string) error
	ChatByRecipient(ctx context.Context, gateID, externalSub string) (*custommodel.Chat, error)
	ChatByID(ctx context.Context, gateID, chatID string) (*custommodel.Chat, error)
}
