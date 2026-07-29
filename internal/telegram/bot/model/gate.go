package model

import (
	"github.com/google/uuid"
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
)

// ProviderType is the provider identifier used to resolve this adapter in the
// registry (internal/provider/registry.go) and to build webhook URLs/paths.
const ProviderType = "telegram_https"

type Gate struct {
	ID            uuid.UUID
	Name          string
	DC            int64
	Bot           *coremodel.Peer
	Enabled       bool
	Token         string
	URI           string
	WebhookSecret string
	CreatedAt     int64
	UpdatedAt     int64
}

type CreateGate struct {
	Name          string
	Token         string
	DC            int64
	Bot           *coremodel.Peer
	Enabled       bool
	URI           string
	WebhookSecret string
}

type UpdateGate struct {
	ID            uuid.UUID
	DC            int64
	Name          *string
	Token         *string
	Bot           *coremodel.Peer
	Enabled       *bool
	WebhookSecret *string
}
