package model

import (
	"github.com/google/uuid"
)

// Interactive represents a rich message with interactive UI elements.
type Interactive struct {
	Body      string
	SingleUse bool
	Markup    *KeyboardMarkup
	ListReply *KeyboardListReply
}

// KeyboardMarkup is a grid of button rows.
type KeyboardMarkup struct {
	Rows []KeyboardRow
}

// KeyboardListReply is a list-style menu with titled sections.
type KeyboardListReply struct {
	MainButtonTitle string
	Sections        []KeyboardRowWithSection
}

// KeyboardRow is a horizontal row of buttons.
type KeyboardRow struct {
	Buttons []KeyboardButton
}

// KeyboardRowWithSection is a labeled group of buttons.
type KeyboardRowWithSection struct {
	Section string
	Buttons []KeyboardButton
}

// KeyboardButton is a single interactive element.
type KeyboardButton struct {
	ID       string
	Label    string
	URL      *KeyboardButtonURL
	Callback *KeyboardButtonCallback
	Request  *KeyboardButtonRequest
}

type KeyboardButtonURL struct {
	URL string
}

type KeyboardButtonCallback struct {
	Data string
}

// KeyboardButtonRequest prompts the user for device data (e.g. location, phone, email).
type KeyboardButtonRequest struct {
	Action string
}

// SendInteractiveCallbackRequest is forwarded when a user clicks an interactive button.
type SendInteractiveCallbackRequest struct {
	From         Peer
	To           Peer
	DomainID     int64
	InReplyTo    string
	ButtonCode   string
	CallbackData string
}

// MessageResponse represents the common return value for provider send operations.
type MessageResponse struct {
	ID string         `json:"id"`
	MD map[string]any `json:"metadata,omitempty"`
}

// Message is the core domain entity representing a message in the system.
type Message struct {
	ID          uuid.UUID    `json:"id"`
	GateID      string       `json:"gate_id"`
	ThreadID    uuid.UUID    `json:"thread_id"`
	DomainID    int64        `json:"domain_id"`
	From        Peer         `json:"from"`
	To          Peer         `json:"to"`
	Text        string       `json:"text"`
	CreatedAt   int64        `json:"created_at"`
	EditedAt    int64        `json:"updated_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Documents   []*Document  `json:"documents,omitempty"`
	Images      []*Image     `json:"images,omitempty"`
	Interactive *Interactive `json:"interactive,omitempty"`
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

// Image defines a visual media attachment.
type Image struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
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
func (i *Image) GetSize() int64      { return i.Size }

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
