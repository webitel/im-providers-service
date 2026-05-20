// Package provider defines the contracts every messaging-platform adapter must satisfy.
// Implementations live in sibling packages (internal/facebook, internal/whatsapp, …).
package provider

import (
	"context"
	"net/url"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

type contextKey string

// WebhookURIKey is the context key for the webhook URI segment injected by the HTTP layer.
const WebhookURIKey contextKey = "webhook_uri"

// Sender is the outbound side of a provider adapter.
type Sender interface {
	// Type returns the provider identifier (e.g. "facebook").
	Type() string
	SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error)
	SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error)
	SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error)
}

// Receiver is the inbound side — it handles raw webhook bytes from the platform.
type Receiver interface {
	Type() string
	HandleWebhook(ctx context.Context, payload []byte) error
}

// Provider groups Sender and Receiver for full-stack adapters.
type Provider interface {
	Sender
	Receiver
}

// Verifier is an optional interface for providers that require a handshake before
// receiving webhooks (e.g. Meta hub.challenge verification).
type Verifier interface {
	Verify(ctx context.Context, query url.Values) (string, error)
}

// SignatureValidator is an optional interface for providers that authenticate
// webhook requests via a cryptographic signature header (e.g. X-Hub-Signature-256).
type SignatureValidator interface {
	ValidateSignature(ctx context.Context, header string, body []byte) error
}
