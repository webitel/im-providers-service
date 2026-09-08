package model

import (
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// ProviderType is the registry key, the gates.type value and the webhook path
// segment. All three must stay identical or outbound sends stop resolving.
const ProviderType = "custom"

const (
	DefaultRequestTimeoutMS = 5000
	DefaultRetryAttempts    = 3
)

type CustomGate struct {
	ID               string                 `json:"id" db:"id"`
	DomainID         int64                  `json:"domain_id" db:"domain_id"`
	Peer             sharedmodel.Peer       `json:"peer" db:"peer"`
	Name             string                 `json:"name" db:"name"`
	CallbackURL      string                 `json:"callback_url" db:"callback_url"`
	AppSecret        string                 `json:"-" db:"app_secret"`
	WebhookURI       string                 `json:"webhook_uri" db:"webhook_uri"`
	AllowedIPs       []string               `json:"allowed_ips" db:"allowed_ips"`
	RequestTimeoutMS int32                  `json:"request_timeout_ms" db:"request_timeout_ms"`
	RetryAttempts    int32                  `json:"retry_attempts" db:"retry_attempts"`
	Status           sharedmodel.GateStatus `json:"status" db:"status"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	Enabled          bool                   `json:"enabled" db:"enabled"`
}

func (g *CustomGate) RequestTimeout() time.Duration {
	if g.RequestTimeoutMS <= 0 {
		return DefaultRequestTimeoutMS * time.Millisecond
	}

	return time.Duration(g.RequestTimeoutMS) * time.Millisecond
}

type CreateCustom struct {
	Name             string
	Dc               int64
	Peer             sharedmodel.Peer
	CallbackURL      string
	AppSecret        string
	AllowedIPs       []string
	RequestTimeoutMS int32
	RetryAttempts    int32
	Enabled          bool
}

type UpdateCustom struct {
	ID               string
	Name             *string
	Peer             *sharedmodel.Peer
	CallbackURL      *string
	AppSecret        *string
	AllowedIPs       *[]string
	RequestTimeoutMS *int32
	RetryAttempts    *int32
	Enabled          *bool
}

func (r UpdateCustom) ApplyTo(g *CustomGate) {
	if r.Name != nil {
		g.Name = *r.Name
	}

	if r.Peer != nil {
		g.Peer = *r.Peer
	}

	if r.CallbackURL != nil && *r.CallbackURL != "" {
		g.CallbackURL = *r.CallbackURL
	}

	if r.AppSecret != nil && *r.AppSecret != "" {
		g.AppSecret = *r.AppSecret
	}

	if r.AllowedIPs != nil {
		g.AllowedIPs = *r.AllowedIPs
	}

	if r.RequestTimeoutMS != nil && *r.RequestTimeoutMS > 0 {
		g.RequestTimeoutMS = *r.RequestTimeoutMS
	}

	if r.RetryAttempts != nil && *r.RetryAttempts >= 0 {
		g.RetryAttempts = *r.RetryAttempts
	}

	if r.Enabled != nil {
		g.Enabled = *r.Enabled
	}
}

// Chat maps a conversation id owned by the external system to the external user
// it belongs to.
type Chat struct {
	GateID      string     `db:"gate_id"`
	ChatID      string     `db:"chat_id"`
	ExternalSub string     `db:"external_sub"`
	ClosedAt    *time.Time `db:"closed_at"`
}
