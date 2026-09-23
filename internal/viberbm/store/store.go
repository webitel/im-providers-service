package store

import (
	"context"

	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// ViberBMStore persists InfoBip Viber BM gateway configuration.
type ViberBMStore interface {
	Insert(ctx context.Context, dc int64, g *vibbmmodel.ViberBMGate) error
	Select(ctx context.Context, id string) (*vibbmmodel.ViberBMGate, error)
	SelectByURI(ctx context.Context, uri string) (*vibbmmodel.ViberBMGate, error)
	SelectBySenderAndURI(ctx context.Context, senderName, uri string) (*vibbmmodel.ViberBMGate, error)
	List(ctx context.Context, dc int64, q string, page, size int) ([]*vibbmmodel.ViberBMGate, bool, error)
	Update(ctx context.Context, g *vibbmmodel.ViberBMGate) error
	Unbind(ctx context.Context, gateID string) error
}
