package store

import (
	"context"

	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

// ViberStore manages persistence for Viber bot gates.
type ViberStore interface {
	// Insert creates the base gate, its bound bot identity, and the Viber configuration.
	Insert(ctx context.Context, dc int64, g *vibmodel.ViberGate) error
	// Select fetches a gate by its internal UUID.
	Select(ctx context.Context, id string) (*vibmodel.ViberGate, error)
	// SelectByURI resolves the gate by its secret webhook path segment (inbound routing).
	SelectByURI(ctx context.Context, uri string) (*vibmodel.ViberGate, error)
	// Update persists changes to an existing gate.
	Update(ctx context.Context, g *vibmodel.ViberGate) error
	// Unbind removes the base gate (cascade drops the Viber configuration and bot).
	Unbind(ctx context.Context, gateID string) error
}
