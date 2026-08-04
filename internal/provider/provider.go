// Package provider defines the contracts every messaging-platform adapter must satisfy.
// Implementations live in sibling packages (internal/facebook, internal/whatsapp, …).
package provider

import (
	"context"
	"net/http"
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

// InteractiveSender is an optional interface for providers that support interactive
// messages (buttons, menus). Implement alongside Sender when the platform allows it.
type InteractiveSender interface {
	SendInteractive(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error)
}

// TypingSender is an optional interface for providers that can forward a native
// "typing…" indicator to the external chat partner (e.g. Telegram
// sendChatAction, Meta sender_action). Providers that do not implement it are a
// silent no-op — the channel simply has no typing indicator.
type TypingSender interface {
	SendTyping(ctx context.Context, req *TypingRequest) error
}

// TypingRequest is a fire-and-forget outbound typing action.
type TypingRequest struct {
	GateID     string
	ExternalID string // recipient's platform-specific id (or internal contact id)
	DomainID   int32
	TypingOn   bool // true = start typing, false = stop
}

// ReactionSender is an optional interface for providers that can set/clear an emoji
// reaction on an existing message (e.g. Telegram setMessageReaction). Non-implementers
// are a silent no-op.
type ReactionSender interface {
	SendReaction(ctx context.Context, req *ReactionRequest) error
}

// ReactionRequest is a fire-and-forget outbound reaction action.
type ReactionRequest struct {
	GateID            string
	ExternalID        string // recipient's platform-specific id
	ExternalMessageID string // platform message id to react to
	DomainID          int32
	Emoji             string // emoji to set; empty when removing
	Removed           bool   // true when removing a reaction
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

// CapabilityReporter is an optional interface for providers that declare
// which delivery-status receipts (delivered/read/failed) their channel
// emits. Channels without it are assumed to support none: messages stay in
// SENT, which is an honest terminal state for such channels.
type CapabilityReporter interface {
	Capabilities() sharedmodel.ProviderCapabilities
}

// Verifier is an optional interface for providers that require a handshake before
// receiving webhooks (e.g. Meta hub.challenge verification).
type Verifier interface {
	Verify(ctx context.Context, query url.Values) (string, error)
}

// SignatureValidator is an optional interface for providers that authenticate
// webhook requests via a header-based signature or secret (e.g. X-Hub-Signature-256,
// X-Telegram-Bot-Api-Secret-Token). Implementations read whatever header(s) they need.
type SignatureValidator interface {
	ValidateSignature(ctx context.Context, headers http.Header, body []byte) error
}
