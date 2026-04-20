package model

import (
	"github.com/google/uuid"
)

// MessageResponse represents the common return value for provider send operations.
type MessageResponse struct {
	ID uuid.UUID      `json:"id"`
	MD map[string]any `json:"metadata,omitempty"`
}

// Message is the core domain entity representing a message in the system.
type Message struct {
	ID        uuid.UUID      `json:"id"`
	ThreadID  uuid.UUID      `json:"thread_id"`
	DomainID  int64          `json:"domain_id"`
	From      Peer           `json:"from"`
	To        Peer           `json:"to"`
	Text      string         `json:"text"`
	CreatedAt int64          `json:"created_at"`
	EditedAt  int64          `json:"updated_at,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Documents []*Document    `json:"documents,omitempty"`
	Images    []*Image       `json:"images,omitempty"`
}

// SendTextRequest defines the payload for sending a plain text message.
type SendTextRequest struct {
	From     Peer   `json:"from"`
	To       Peer   `json:"to"`
	Body     string `json:"body"`
	DomainID int64  `json:"domain_id"`
}

// SendTextResponse confirms the delivery of a text message.
type SendTextResponse struct {
	ID uuid.UUID `json:"id"`
	To Peer      `json:"to"`
}
