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
	// GetTypeByID returns the provider type for a gate by its UUID.
	GetTypeByID(ctx context.Context, id string) (sharedmodel.GateType, error)
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
	// GetLocale returns the BCP-47-style locale string (e.g. "uk_UA") stored for a user.
	// Returns ErrNotFound when no locale was previously recorded.
	GetLocale(ctx context.Context, gateID, userID string) (string, error)
	// SetLocale persists the user locale so it can be used for template selection.
	SetLocale(ctx context.Context, gateID, userID, locale string) error
}

// TemplateStore resolves and manages gate-specific system message template overrides.
// Returning ErrNotFound signals "no override — use built-in default".
type TemplateStore interface {
	// GetTemplate returns the template string for the given gate and event type.
	// Returns ErrNotFound when no gate-specific override is configured.
	GetTemplate(ctx context.Context, gateID, eventType string) (string, error)
	// SetTemplate creates or replaces the template override for (gateID, eventType).
	SetTemplate(ctx context.Context, gateID, eventType, template string, domainID int64) error
	// DeleteTemplate removes the override. Returns ErrNotFound if none exists.
	DeleteTemplate(ctx context.Context, gateID, eventType string) error
	// ListTemplates returns all overrides configured for the given gate.
	ListTemplates(ctx context.Context, gateID string) ([]TemplateRow, error)
}

// TemplateRow is a single row from gate_message_templates.
type TemplateRow struct {
	GateID    string
	EventType string
	Template  string
}
