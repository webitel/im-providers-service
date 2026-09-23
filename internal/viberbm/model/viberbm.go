package model

import (
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// ViberBMGate is a configured InfoBip Viber Business Messages gateway.
// It is distinct from the direct Viber Bot gate (internal/viber): recipients
// are MSISDNs and traffic flows through a per-account InfoBip host.
type ViberBMGate struct {
	ID            string                 `json:"id" db:"id"`
	DomainID      int64                  `json:"domain_id" db:"domain_id"`
	Peer          sharedmodel.Peer       `json:"peer" db:"peer"`
	Name          string                 `json:"name" db:"name"`
	BaseURL       string                 `json:"base_url" db:"base_url"`
	SenderName    string                 `json:"sender_name" db:"sender_name"`
	APIKey        string                 `json:"-" db:"api_key"`
	WebhookSecret string                 `json:"-" db:"webhook_secret"`
	WebhookURI    string                 `json:"webhook_uri" db:"webhook_uri"`
	Status        sharedmodel.GateStatus `json:"status" db:"status"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
	Enabled       bool                   `json:"enabled" db:"enabled"`
}

// CreateViberBM is the input to create a Viber BM gateway.
type CreateViberBM struct {
	Name          string
	Dc            int64
	Peer          sharedmodel.Peer
	BaseURL       string
	SenderName    string
	APIKey        string
	WebhookSecret string
	Enabled       bool
}

// UpdateViberBM is the partial-update input for a Viber BM gateway.
// A non-empty APIKey / WebhookSecret rotates the stored secret.
type UpdateViberBM struct {
	ID            string
	Name          *string
	Peer          *sharedmodel.Peer
	BaseURL       *string
	SenderName    *string
	APIKey        *string
	WebhookSecret *string
	Enabled       *bool
}

func (r UpdateViberBM) ApplyTo(g *ViberBMGate) {
	if r.Name != nil {
		g.Name = *r.Name
	}

	if r.Peer != nil {
		g.Peer = *r.Peer
	}

	if r.BaseURL != nil && *r.BaseURL != "" {
		g.BaseURL = *r.BaseURL
	}

	if r.SenderName != nil && *r.SenderName != "" {
		g.SenderName = *r.SenderName
	}

	// Empty secret means "keep existing" — never overwrite with a blank,
	// otherwise a partial update silently erases the stored credential.
	if r.APIKey != nil && *r.APIKey != "" {
		g.APIKey = *r.APIKey
	}

	if r.WebhookSecret != nil && *r.WebhookSecret != "" {
		g.WebhookSecret = *r.WebhookSecret
	}

	if r.Enabled != nil {
		g.Enabled = *r.Enabled
	}
}
