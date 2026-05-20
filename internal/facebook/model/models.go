package model

import (
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// MetaApp represents the parent OAuth application configuration shared by
// Facebook and WhatsApp gates.
type MetaApp struct {
	ID               string    `json:"id" db:"id"`
	URI              string    `json:"uri" db:"uri"`
	Name             string    `json:"name" db:"name"`
	AppID            string    `json:"app_id" db:"app_id"`
	AppSecret        string    `json:"-" db:"app_secret"`
	VerifyToken      string    `json:"-" db:"verify_token"`
	OAuthRedirectURI string    `json:"oauth_redirect_uri" db:"redirect_uri"`
	Scopes           []string  `json:"scopes" db:"scopes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// FacebookGate represents a Facebook Page gate configuration.
type FacebookGate struct {
	ID        string               `json:"id" db:"id"`
	DomainID  int64                `json:"domain_id" db:"domain_id"`
	Peer      sharedmodel.Peer     `json:"peer" db:"peer"`
	Name      string               `json:"name" db:"name"`
	MetaAppID string               `json:"meta_app_id" db:"meta_app_id"`
	PageID    string               `json:"page_id" db:"page_id"`
	PageName  string               `json:"page_name" db:"page_name"`
	PageToken string               `json:"-" db:"page_token"`
	Webhook   string               `json:"webhook" db:"webhook"`
	Status    sharedmodel.GateStatus `json:"status" db:"status"`
	CreatedAt time.Time            `json:"created_at" db:"created_at"`
	UpdatedAt time.Time            `json:"updated_at" db:"updated_at"`
	Enabled   bool                 `json:"enabled" db:"enabled"`
}

// CreateMetaApp is the request payload for creating a new Meta OAuth application.
type CreateMetaApp struct {
	Name             string
	URI              string
	AppID            string
	AppSecret        string
	OAuthRedirectURI string
	Scopes           []string
	VerifyToken      string
}

// UpdateMetaApp is the request payload for updating an existing Meta OAuth application.
type UpdateMetaApp struct {
	ID               string
	Name             *string
	AppSecret        *string
	OAuthRedirectURI *string
	Scopes           []string
	VerifyToken      *string
}

// OAuthStart is the request payload for initiating the Meta OAuth flow.
type OAuthStart struct {
	MetaAppID   string
	ExtraScopes []string
}

// OAuthCallback is the request payload for completing the Meta OAuth flow.
type OAuthCallback struct {
	MetaAppID string
	Code      string
	State     string
}

// CreateFacebook is the request payload for creating a new Facebook Page gate.
type CreateFacebook struct {
	Name      string
	Dc        int64
	MetaAppID string
	PageID    string
	PageToken string
	Peer      sharedmodel.Peer
	Enabled   bool
}

// UpdateFacebook is the request payload for updating an existing Facebook Page gate.
type UpdateFacebook struct {
	ID        string
	Name      *string
	PageToken *string
	Enabled   *bool
	Peer      *sharedmodel.Peer
}
