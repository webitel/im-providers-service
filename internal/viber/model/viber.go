package model

import (
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

type ViberGate struct {
	ID           string                 `json:"id" db:"id"`
	DomainID     int64                  `json:"domain_id" db:"domain_id"`
	Peer         sharedmodel.Peer       `json:"peer" db:"peer"`
	Name         string                 `json:"name" db:"name"`
	BotID        string                 `json:"bot_id" db:"bot_id"`
	BotURI       string                 `json:"bot_uri" db:"bot_uri"`
	AuthToken    string                 `json:"-" db:"auth_token"`
	SenderName   string                 `json:"sender_name" db:"sender_name"`
	SenderAvatar string                 `json:"sender_avatar" db:"sender_avatar"`
	WebhookURI   string                 `json:"webhook_uri" db:"webhook_uri"`
	Status       sharedmodel.GateStatus `json:"status" db:"status"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	Enabled      bool                   `json:"enabled" db:"enabled"`
}

type AccountInfo struct {
	ID     string
	Name   string
	URI    string
	Avatar string
}

type CreateViber struct {
	Name         string
	Dc           int64
	Peer         sharedmodel.Peer
	AuthToken    string
	SenderName   string
	SenderAvatar string
	BotID        string
	BotURI       string
	WebhookURI   string
	Enabled      bool
}

type UpdateViber struct {
	ID           string
	Name         *string
	Peer         *sharedmodel.Peer
	AuthToken    *string
	SenderName   *string
	SenderAvatar *string
	Enabled      *bool
}

func (r UpdateViber) ApplyTo(g *ViberGate) {
	if r.Name != nil {
		g.Name = *r.Name
	}
	if r.Peer != nil {
		g.Peer = *r.Peer
	}
	if r.AuthToken != nil && *r.AuthToken != "" {
		g.AuthToken = *r.AuthToken
	}
	if r.SenderName != nil {
		g.SenderName = *r.SenderName
	}
	if r.SenderAvatar != nil {
		g.SenderAvatar = *r.SenderAvatar
	}
	if r.Enabled != nil {
		g.Enabled = *r.Enabled
	}
}
