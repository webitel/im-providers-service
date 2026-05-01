package store

import (
	"context"
	"errors"

	"github.com/webitel/im-providers-service/internal/domain/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("data conflict or optimistic lock failure")
)

// Store aggregates all specialized interfaces into one unit.
// This allows passing the entire storage layer as a single dependency.
type Store interface {
	Gates() GateStore
	Meta() MetaAppStore
	Facebook() FacebookStore
	WhatsApp() WhatsAppStore
}

// GateStore handles high-level summary operations for the dashboard.
type GateStore interface {
	// List returns a paginated list of all gates across all providers.
	List(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error)
	// Delete removes the base gate and its specific configuration (via cascade).
	Delete(ctx context.Context, id string) error
}

// MetaAppStore manages shared technical credentials for Meta API.
type MetaAppStore interface {
	Insert(ctx context.Context, a *model.MetaApp) error
	Select(ctx context.Context, id string) (*model.MetaApp, error)
	Update(ctx context.Context, a *model.MetaApp) error
	Delete(ctx context.Context, id string) error
}

// FacebookStore manages logic for Facebook Page integrations.
type FacebookStore interface {
	// Insert creates a logic gate and links it to a MetaApp.
	Insert(ctx context.Context, dc int64, g *model.FacebookGate) error
	Select(ctx context.Context, id string) (*model.FacebookGate, error)
	SelectByPageAndURI(ctx context.Context, pageID, uri string) (*model.FacebookGate, error)
	Update(ctx context.Context, g *model.FacebookGate) error
	Unbind(ctx context.Context, gateID string) error
}

// WhatsAppStore manages logic for WhatsApp Business API integrations.
type WhatsAppStore interface {
	Insert(ctx context.Context, g *model.WhatsAppGate) error
	Select(ctx context.Context, id string) (*model.WhatsAppGate, error)
	Update(ctx context.Context, g *model.WhatsAppGate) error
}

// GateState holds minimal data for fast webhook routing and filtering.
// This structure is cached to avoid frequent database hits during high-traffic webhooks.
type GateState struct {
	// GateID is the internal system UUID for the gate.
	// For external platforms, this represents:
	// - Facebook:  Page ID
	// - Instagram: Scoped Business ID
	// - WhatsApp:  Phone Number ID (WABA)
	// - Telegram:  Bot Token Hash / Bot ID
	// - Viber:      Bot Token Hash / Bot ID
	GateID  string
	Enabled bool
	Issuer  string
	Sub     string
	Domain  int64
}

// GateCache defines the contract for high-speed gate lookups across all providers.
type GateCache interface {
	// Set stores a gate's state using a unique provider key (e.g., PageID, PhoneID).
	Set(key string, state GateState)
	// Get retrieves cached gate state by its provider key.
	Get(key string) (GateState, bool)
	// Delete removes a gate's state from the cache (used on updates/deletion).
	Delete(key string)
}

// ExternalUserCache defines the contract for identity tracking
type ExternalUserCache interface {
	IsKnown(ctx context.Context, user *model.ExternalUser) (bool, error)
	MarkKnown(ctx context.Context, user *model.ExternalUser) error
}
