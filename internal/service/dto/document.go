package dto

import (
	"github.com/google/uuid"
	"github.com/webitel/im-providers-service/internal/domain/model"
)

type Document struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
}

type DocumentRequest struct {
	Body      string      `json:"body"`
	Documents []*Document `json:"documents"`
}

type SendDocumentRequest struct {
	From     model.Peer      `json:"from"`
	To       model.Peer      `json:"to"`
	Document DocumentRequest `json:"document"`
	DomainID int64           `json:"domain_id"`
}

type SendDocumentResponse struct {
	ID uuid.UUID  `json:"id"`
	To model.Peer `json:"to"`
}

// Getters and Setters for Document
func (d *Document) GetID() int64        { return d.ID }
func (d *Document) GetURL() string      { return d.URL }
func (d *Document) GetMimeType() string { return d.MimeType }
func (d *Document) GetName() string     { return d.Name }
func (d *Document) SetID(id int64)      { d.ID = id }
func (d *Document) SetMime(mime string) { d.MimeType = mime }
func (d *Document) SetName(name string) { d.Name = name }
