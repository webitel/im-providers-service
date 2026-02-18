package facebook

import (
	"context"

	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/im-providers-service/internal/service/dto"
)

// [INTERFACE_GUARDS] Ensure the adapter satisfies all required domain contracts.
var (
	_ provider.Provider = (*facebookProvider)(nil)
	_ provider.Sender   = (*facebookProvider)(nil)
	_ provider.Receiver = (*facebookProvider)(nil)
)

type facebookProvider struct {
	client *Client // Internal Meta Graph API client
}

// New creates an initialized instance of the Facebook adapter.
// [DI] Injecting Messenger service to handle inbound events from webhooks.
func New(messenger service.Messenger) provider.Provider {
	return &facebookProvider{
		client: NewClient(messenger),
	}
}

// Type returns the provider unique identifier.
func (p *facebookProvider) Type() string { return "facebook" }

// --- [RECEIVER_IMPLEMENTATION] ---

func (p *facebookProvider) HandleWebhook(ctx context.Context, payload []byte) error {
	// [DELEGATION] Forward raw events to the low-level client for parsing.
	return p.client.ProcessEvent(ctx, payload)
}

// --- [SENDER_IMPLEMENTATION] ---

func (p *facebookProvider) SendText(ctx context.Context, req *dto.SendTextRequest) (*dto.SendTextResponse, error) {
	return p.client.SendTextMessage(ctx, req)
}

func (p *facebookProvider) SendImage(ctx context.Context, req *dto.SendImageRequest) (*dto.SendImageResponse, error) {
	return p.client.SendImageMessage(ctx, req)
}

func (p *facebookProvider) SendDocument(ctx context.Context, req *dto.SendDocumentRequest) (*dto.SendDocumentResponse, error) {
	return p.client.SendDocumentMessage(ctx, req)
}
