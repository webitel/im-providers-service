package store

import (
	"context"

	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

// InstagramStore manages logic for Instagram Business Account integrations.
type InstagramStore interface {
	// Insert creates a gate and links it to a MetaApp.
	Insert(ctx context.Context, dc int64, g *igmodel.InstagramGate) error
	Select(ctx context.Context, id string) (*igmodel.InstagramGate, error)
	SelectByBusinessAccountAndURI(ctx context.Context, businessAccountID, uri string) (*igmodel.InstagramGate, error)
	Update(ctx context.Context, g *igmodel.InstagramGate) error
	Unbind(ctx context.Context, gateID string) error
}

// MetaAppStore is re-exported from facebook for convenience.
type MetaAppStore = fbstore.MetaAppStore

// GateCacheKey is the shared GateCache key for an Instagram gate, keyed by the
// business account id and namespaced by provider. The provider (set/get) and the
// store (invalidate on update/unbind) MUST use this same key, or invalidation
// silently no-ops and a stale gate keeps being served until the LRU/TTL expires.
func GateCacheKey(businessAccountID string) string {
	return "instagram:" + businessAccountID
}
