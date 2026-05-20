package store

import (
	"context"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
)

// FacebookStore manages logic for Facebook Page integrations.
type FacebookStore interface {
	// Insert creates a gate and links it to a MetaApp.
	Insert(ctx context.Context, dc int64, g *fbmodel.FacebookGate) error
	Select(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
	SelectByPageAndURI(ctx context.Context, pageID, uri string) (*fbmodel.FacebookGate, error)
	Update(ctx context.Context, g *fbmodel.FacebookGate) error
	Unbind(ctx context.Context, gateID string) error
}

// MetaAppStore manages shared technical credentials for the Meta API.
type MetaAppStore interface {
	Insert(ctx context.Context, a *fbmodel.MetaApp) error
	Select(ctx context.Context, id string) (*fbmodel.MetaApp, error)
	SelectByURI(ctx context.Context, uri string) (*fbmodel.MetaApp, error)
	Update(ctx context.Context, a *fbmodel.MetaApp) error
	Delete(ctx context.Context, id string) error
}
