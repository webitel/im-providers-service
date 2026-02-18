package facebook

import (
	"context"

	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/im-providers-service/internal/service/dto"
)

// Client handles low-level communication with Meta Graph API and dispatches inbound events.
type Client struct {
	messenger service.Messenger // [CORE_LINK] Bridge to Webitel internal logic
	// [TODO] Add http.Client, BaseURL, and Auth tokens.
}

// NewClient initializes a new Facebook API client with mandatory messenger service.
func NewClient(messenger service.Messenger) *Client {
	return &Client{
		messenger: messenger,
	}
}

// --- [API_OUTBOUND_METHODS] ---

// SendTextMessage triggers the Meta Graph API to deliver text.
func (c *Client) SendTextMessage(ctx context.Context, req *dto.SendTextRequest) (*dto.SendTextResponse, error) {
	panic("unimplemented: [META_API] POST /me/messages (text)")
}

// SendImageMessage triggers the Meta Graph API to deliver image media.
func (c *Client) SendImageMessage(ctx context.Context, req *dto.SendImageRequest) (*dto.SendImageResponse, error) {
	panic("unimplemented: [META_API] POST /me/messages (image)")
}

// SendDocumentMessage triggers the Meta Graph API to deliver files.
func (c *Client) SendDocumentMessage(ctx context.Context, req *dto.SendDocumentRequest) (*dto.SendDocumentResponse, error) {
	panic("unimplemented: [META_API] POST /me/messages (document)")
}

// --- [WEBHOOK_INBOUND_LOGIC] ---

// ProcessEvent parses raw JSON from webhook and dispatches it to the Messenger service.
func (c *Client) ProcessEvent(ctx context.Context, payload []byte) error {
	// [FLOW]
	// 1. Unmarshal Meta JSON
	// 2. Map to internal dto.InboundMessage (or SendTextRequest for logic reuse)
	// 3. Call c.messenger.SendText (or specific Inbound method)
	panic("unimplemented: [WEBHOOK_PARSER] facebook payload analysis and dispatch to messenger")
}
