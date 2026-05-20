package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/im-providers-service/pkg/crypto"
	wamodel "github.com/webitel/im-providers-service/internal/whatsapp/model"
)

type whatsAppStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

func NewWhatsAppStore(pool *pgxpool.Pool, crypt crypto.Encryptor) WhatsAppStore {
	return &whatsAppStore{
		pool:   pool,
		crypto: crypt,
	}
}

// Insert implements [WhatsAppStore].
func (w *whatsAppStore) Insert(ctx context.Context, g *wamodel.WhatsAppGate) error {
	panic("unimplemented")
}

// Select implements [WhatsAppStore].
func (w *whatsAppStore) Select(ctx context.Context, id string) (*wamodel.WhatsAppGate, error) {
	panic("unimplemented")
}

// Update implements [WhatsAppStore].
func (w *whatsAppStore) Update(ctx context.Context, g *wamodel.WhatsAppGate) error {
	panic("unimplemented")
}
