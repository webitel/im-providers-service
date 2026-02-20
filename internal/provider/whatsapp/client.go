package whatsapp

import (
	"context"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/service"
)

// Client encapsulates low-level communication with WhatsApp Business API.
type Client struct {
	messenger service.Messenger // [CORE_LINK] Bridge to pass inbound messages to Webitel
	// [TODO] Add http.Client, WhatsApp Cloud API tokens, and PhoneID.
}

// NewClient initializes a new WhatsApp API client.
func NewClient(messenger service.Messenger) *Client {
	return &Client{
		messenger: messenger,
	}
}

// --- [API_OUTBOUND_METHODS] ---

// SendTextMessage triggers the WhatsApp Cloud API to deliver text.
func (c *Client) SendTextMessage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	panic("unimplemented: [WHATSAPP_API] POST /messages (text)")
}

// SendImageMessage triggers the WhatsApp Cloud API to deliver image media.
func (c *Client) SendImageMessage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	panic("unimplemented: [WHATSAPP_API] POST /messages (image)")
}

// SendDocumentMessage triggers the WhatsApp Cloud API to deliver files/documents.
func (c *Client) SendDocumentMessage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	panic("unimplemented: [WHATSAPP_API] POST /messages (document)")
}

// --- [WEBHOOK_INBOUND_LOGIC] ---

// ProcessEvent parses raw WhatsApp JSON and converts it to unified internal events.
func (c *Client) ProcessEvent(ctx context.Context, payload []byte) error {
	// [FLOW]
	// 1. Unmarshal WhatsApp Cloud API JSON (messages, statuses, errors)
	// 2. Map to internal DTOs
	// 3. Dispatch to c.messenger for core processing
	panic("unimplemented: [WHATSAPP_PARSER] complex cloud api payload analysis and dispatch")
}
