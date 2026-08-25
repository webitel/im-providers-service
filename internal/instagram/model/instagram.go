package model

import (
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// InstagramGate represents an Instagram Business Account gate configuration.
type InstagramGate struct {
	ID                string                 `json:"id" db:"id"`
	DomainID          int64                  `json:"domain_id" db:"domain_id"`
	Peer              sharedmodel.Peer       `json:"peer" db:"peer"`
	Name              string                 `json:"name" db:"name"`
	MetaAppID         string                 `json:"meta_app_id" db:"meta_app_id"`
	BusinessAccountID string                 `json:"business_account_id" db:"business_account_id"`
	IGToken           string                 `json:"-" db:"ig_access_token"`
	Status            sharedmodel.GateStatus `json:"status" db:"status"`
	Enabled           bool                   `json:"enabled" db:"enabled"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

// CreateInstagram is the request payload for creating a new Instagram Business Account gate.
type CreateInstagram struct {
	Name              string
	Dc                int64
	MetaAppID         string
	BusinessAccountID string
	IGToken           string
	Peer              sharedmodel.Peer
	Enabled           bool
}

// UpdateInstagram is the request payload for updating an existing Instagram Business Account gate.
// Dc is the caller's domain, used to scope the update to gates the caller owns.
type UpdateInstagram struct {
	ID      string
	Dc      int64
	Name    *string
	IGToken *string
	Enabled *bool
	Peer    *sharedmodel.Peer
}

func (r UpdateInstagram) ApplyTo(gate *InstagramGate) {
	if r.Name != nil {
		gate.Name = *r.Name
	}

	if r.Enabled != nil {
		gate.Enabled = *r.Enabled
	}

	if r.IGToken != nil {
		gate.IGToken = *r.IGToken
	}

	if r.Peer != nil {
		gate.Peer = *r.Peer
	}
}

// OAuthStart initiates Instagram Business Login OAuth.
type OAuthStart struct {
	MetaAppID string
}

// OAuthCallback carries the OAuth response from Instagram.
type OAuthCallback struct {
	MetaAppID string
	Code      string
	State     string
}
