package model

import (
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// FacebookGate represents a Facebook Page gate configuration.
type FacebookGate struct {
	ID        string                 `json:"id" db:"id"`
	DomainID  int64                  `json:"domain_id" db:"domain_id"`
	Peer      sharedmodel.Peer       `json:"peer" db:"peer"`
	Name      string                 `json:"name" db:"name"`
	MetaAppID string                 `json:"meta_app_id" db:"meta_app_id"`
	PageID    string                 `json:"page_id" db:"page_id"`
	PageName  string                 `json:"page_name" db:"page_name"`
	PageToken string                 `json:"-" db:"page_token"`
	Webhook   string                 `json:"webhook" db:"webhook"`
	Status    sharedmodel.GateStatus `json:"status" db:"status"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
	Enabled   bool                   `json:"enabled" db:"enabled"`
}

type CreateFacebook struct {
	Name      string
	Dc        int64
	MetaAppID string
	PageID    string
	PageToken string
	Peer      sharedmodel.Peer
	Enabled   bool
}

type UpdateFacebook struct {
	ID        string
	Name      *string
	PageToken *string
	Enabled   *bool
	Peer      *sharedmodel.Peer
}

func (r UpdateFacebook) ApplyTo(gate *FacebookGate) {
	if r.Name != nil {
		gate.Name = *r.Name
	}
	if r.Enabled != nil {
		gate.Enabled = *r.Enabled
	}
	if r.PageToken != nil {
		gate.PageToken = *r.PageToken
	}
	if r.Peer != nil {
		gate.Peer = *r.Peer
	}
}

// MenuItem represents a single entry in the Messenger Persistent Menu.
// Exactly one of Payload, URL, or Nested should be set.
type MenuItem struct {
	Title   string
	Payload string     // postback button
	URL     string     // web_url button
	Nested  []MenuItem // nested submenu (one level deep)
}
