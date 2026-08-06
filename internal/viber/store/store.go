package store

import (
	"context"

	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

type ViberStore interface {
	Insert(ctx context.Context, dc int64, g *vibmodel.ViberGate) error
	Select(ctx context.Context, id string) (*vibmodel.ViberGate, error)
	SelectByURI(ctx context.Context, uri string) (*vibmodel.ViberGate, error)
	Update(ctx context.Context, g *vibmodel.ViberGate) error
	Unbind(ctx context.Context, gateID string) error
}
