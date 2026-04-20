package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

type whatsAppStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

func NewWhatsAppStore(pool *pgxpool.Pool, crypt crypto.Encryptor) store.WhatsAppStore {
	return &whatsAppStore{
		pool:   pool,
		crypto: crypt,
	}
}

// Insert implements [store.WhatsAppStore].
func (w *whatsAppStore) Insert(ctx context.Context, g *model.WhatsAppGate) error {
	panic("unimplemented")
}

// Select implements [store.WhatsAppStore].
func (w *whatsAppStore) Select(ctx context.Context, id string) (*model.WhatsAppGate, error) {
	panic("unimplemented")
}

// Update implements [store.WhatsAppStore].
func (w *whatsAppStore) Update(ctx context.Context, g *model.WhatsAppGate) error {
	panic("unimplemented")
}
