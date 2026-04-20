package model

import "time"

type GateSummary struct {
	ID            string     `db:"id"`
	Name          string     `db:"name"`
	Type          GateType   `db:"type"`
	Status        GateStatus `db:"status"`
	WebhookURL    string     `db:"-"`
	Contact       string     `db:"contact"`
	ProviderAppID *string    `db:"provider_app_id"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

// MetaApp represents the parent OAuth configuration.
type MetaApp struct {
	ID               string    `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	AppID            string    `json:"app_id" db:"app_id"`
	AppSecret        string    `json:"-" db:"app_secret"`
	SystemUserToken  string    `json:"-" db:"system_user_token"`
	OAuthRedirectURI string    `json:"oauth_redirect_uri" db:"redirect_uri"`
	Scopes           []string  `json:"scopes" db:"scopes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type FacebookGate struct {
	ID        string     `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	MetaAppID string     `json:"meta_app_id" db:"meta_app_id"`
	PageID    string     `json:"page_id" db:"page_id"`
	PageName  string     `json:"page_name" db:"page_name"`
	PageToken string     `json:"-" db:"page_token"` // Used for encrypted token storage
	Webhook   string     `json:"webhook" db:"webhook"`
	Status    GateStatus `json:"status" db:"status"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	Enabled   bool       `json:"enabled" db:"enabled"`
}

type WhatsAppGate struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	MetaAppID     string     `json:"meta_app_id"`
	WABAID        string     `json:"waba_id"`
	PhoneNumberID string     `json:"phone_number_id"`
	PhoneDisplay  string     `json:"phone_display"`
	AccessToken   string     `json:"-"`
	Status        GateStatus `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
