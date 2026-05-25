package model

import "time"

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

type CreateMetaApp struct {
	Name             string
	URI              string
	AppID            string
	AppSecret        string
	OAuthRedirectURI string
	Scopes           []string
	VerifyToken      string
}

type UpdateMetaApp struct {
	ID               string
	Name             *string
	AppSecret        *string
	OAuthRedirectURI *string
	Scopes           []string
	VerifyToken      *string
}

func (r UpdateMetaApp) ApplyTo(app *MetaApp) {
	if r.Name != nil {
		app.Name = *r.Name
	}
	if r.AppSecret != nil {
		app.AppSecret = *r.AppSecret
	}
	if r.OAuthRedirectURI != nil {
		app.OAuthRedirectURI = *r.OAuthRedirectURI
	}
	if r.Scopes != nil {
		app.Scopes = r.Scopes
	}
	if r.VerifyToken != nil {
		app.VerifyToken = *r.VerifyToken
	}
}
