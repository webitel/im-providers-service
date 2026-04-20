package model

import (
	"github.com/google/uuid"
)

// Document defines a generic file attachment.
type Document struct {
	ID       string `json:"id"` // Internal or provider-specific ID
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	URL      string `json:"url,omitempty"`
}

// DocumentRequest wraps the message body and associated files.
type DocumentRequest struct {
	Body      string      `json:"body"`
	Documents []*Document `json:"documents"`
}

// SendDocumentRequest is the payload for sending one or more documents.
type SendDocumentRequest struct {
	From     Peer            `json:"from"`
	To       Peer            `json:"to"`
	Document DocumentRequest `json:"document"`
	DomainID int64           `json:"domain_id"`
}

// SendDocumentResponse confirms the document was sent.
type SendDocumentResponse struct {
	ID uuid.UUID `json:"id"`
	To Peer      `json:"to"`
}

// Getters for Document to satisfy provider interfaces.
func (d *Document) GetID() string       { return d.ID }
func (d *Document) GetURL() string      { return d.URL }
func (d *Document) GetMimeType() string { return d.MimeType }
func (d *Document) GetName() string     { return d.FileName }
