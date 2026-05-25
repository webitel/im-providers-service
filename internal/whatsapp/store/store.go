package store

import (
	"context"

	wamodel "github.com/webitel/im-providers-service/internal/whatsapp/model"
)

// WhatsAppStore manages logic for WhatsApp Business API integrations.
type WhatsAppStore interface {
	Insert(ctx context.Context, g *wamodel.WhatsAppGate) error
	Select(ctx context.Context, id string) (*wamodel.WhatsAppGate, error)
	Update(ctx context.Context, g *wamodel.WhatsAppGate) error
}
