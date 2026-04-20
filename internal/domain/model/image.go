package model

import (
	"github.com/google/uuid"
)

// Image defines a visual media attachment.
type Image struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url"`
}

// ImageRequest wraps images and an optional caption.
type ImageRequest struct {
	Images []*Image `json:"images"`
	Body   string   `json:"body"`
}

// SendImageRequest is the payload for sending visual content.
type SendImageRequest struct {
	From     Peer         `json:"from"`
	To       Peer         `json:"to"`
	Image    ImageRequest `json:"image"`
	DomainID int64        `json:"domain_id"`
}

// SendImageResponse confirms the image was sent.
type SendImageResponse struct {
	ID uuid.UUID `json:"id"`
	To Peer      `json:"to"`
}

// Getters for Image for idiomatic access.
func (i *Image) GetID() string       { return i.ID }
func (i *Image) GetURL() string      { return i.URL }
func (i *Image) GetMimeType() string { return i.MimeType }
func (i *Image) GetName() string     { return i.FileName }
