package provider

import (
	"context"
	"net/url"

	"github.com/webitel/im-providers-service/internal/domain/model"
)

type contextKey string

// WebhookURIKey is the context key for the webhook URI segment.
const WebhookURIKey contextKey = "webhook_uri"

// Sender defines the contract for outgoing communication.
type Sender interface {
	// Type returns the provider identifier (e.g., "facebook").
	Type() string
	SendText(ctx context.Context, req *model.Message) (*model.MessageResponse, error)
	SendImage(ctx context.Context, req *model.Message) (*model.MessageResponse, error)
	SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error)
}

// Receiver defines the contract for incoming communication (Webhooks).
// [INTERFACE_SEGREGATION] Focuses only on processing events FROM external platforms.
type Receiver interface {
	// Type returns the provider identifier.
	Type() string
	HandleWebhook(ctx context.Context, payload []byte) error
}

// Provider groups both behaviors for full-stack adapters.
// [COMPOSITE_INTERFACE] Most of your adapters (FB, WA) will implement this.
type Provider interface {
	Sender
	Receiver
}

// Verifier is an optional interface for providers that require
// handshake verification (e.g., Meta/Facebook hub challenge).
type Verifier interface {
	Verify(ctx context.Context, query url.Values) (string, error)
}

// SignatureValidator is an optional interface for providers that validate
// request authenticity via a cryptographic signature header (e.g., X-Hub-Signature-256).
type SignatureValidator interface {
	ValidateSignature(ctx context.Context, header string, body []byte) error
}
