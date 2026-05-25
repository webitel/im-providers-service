package store

import (
	"context"
	"errors"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("data conflict or optimistic lock failure")
)

// GateStore handles high-level summary operations for the dashboard.
type GateStore interface {
	// List returns a paginated list of all gates across all providers.
	List(ctx context.Context, f sharedmodel.ListFilter) ([]*sharedmodel.GateSummary, bool, error)
	// Delete removes the base gate and its specific configuration (via cascade).
	Delete(ctx context.Context, id string) error
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
	IsKnown(ctx context.Context, user *sharedmodel.ExternalUser) (bool, error)
	MarkKnown(ctx context.Context, user *sharedmodel.ExternalUser) error
}
