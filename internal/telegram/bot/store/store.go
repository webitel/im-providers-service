package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

type TelegramBotStore interface {
	Insert(context.Context, *model.CreateGate) (*model.Gate, error)
	Get(context.Context, *model.GetGateFilters) (*model.Gate, error)
	Update(context.Context, *model.UpdateGate) (*model.Gate, error)
	Delete(context.Context, uuid.UUID, int64) error
}
