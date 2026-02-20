package model

import (
	"github.com/google/uuid"
)

// Response struct for Provider [WHATSAPP/FACEBOOK/TELEGRAM/VIBER] Send operations.
type MessageResponse struct {
	ID uuid.UUID      `json:"id"`
	MD map[string]any `json:"metadata,omitempty"`
}

type (
	Message struct {
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

	Document struct {
		ID       string `json:"id"`
		FileName string `json:"file_name"`
		MimeType string `json:"mime_type"`
		Size     int64  `json:"size"`
	}

	Image struct {
		ID       string `json:"id"`
		FileName string `json:"file_name"`
		MimeType string `json:"mime_type"`
		URL      string `json:"url"`
	}
)
