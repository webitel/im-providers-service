package whatsapp

import (
	"context"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/service"
)

// [INTERFACE_GUARDS] Ensure the adapter strictly adheres to all provider contracts.
var (
	_ provider.Provider = (*whatsAppProvider)(nil)
	_ provider.Sender   = (*whatsAppProvider)(nil)
	_ provider.Receiver = (*whatsAppProvider)(nil)
)

type whatsAppProvider struct {
	client *Client // Internal WhatsApp Business API client
}

// New creates an initialized instance of the WhatsApp adapter.
// [DI] Injecting Messenger service to handle inbound events from WhatsApp webhooks.
func New(messenger service.Messenger) provider.Provider {
	return &whatsAppProvider{
		client: NewClient(messenger),
	}
}

// Type returns the provider unique identifier.
func (p *whatsAppProvider) Type() string { return "whatsapp" }

// --- [RECEIVER_IMPLEMENTATION] ---

func (p *whatsAppProvider) HandleWebhook(ctx context.Context, payload []byte) error {
	// [DELEGATION] Forward raw WhatsApp JSON events to the specialized client.
	return p.client.ProcessEvent(ctx, payload)
}

// --- [SENDER_IMPLEMENTATION] ---
func (p *whatsAppProvider) SendText(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	// [MAPPING] Map domain DTO to WhatsApp Business API text message.
	return p.client.SendTextMessage(ctx, req)
}

func (p *whatsAppProvider) SendImage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	// [MAPPING] Handle media-based messages for WhatsApp.
	return p.client.SendImageMessage(ctx, req)
}

func (p *whatsAppProvider) SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	// [MAPPING] Deliver PDF/Documents via WhatsApp Cloud API.
	return p.client.SendDocumentMessage(ctx, req)
}
