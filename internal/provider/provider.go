package provider

import (
	"context"

	"github.com/webitel/im-providers-service/internal/service/dto"
)

// Sender defines the contract for outgoing communication.
// [INTERFACE_SEGREGATION] Focuses only on delivering messages TO external platforms.
type Sender interface {
	// Type returns the provider identifier (e.g., "facebook").
	Type() string
	SendText(ctx context.Context, req *dto.SendTextRequest) (*dto.SendTextResponse, error)
	SendImage(ctx context.Context, req *dto.SendImageRequest) (*dto.SendImageResponse, error)
	SendDocument(ctx context.Context, req *dto.SendDocumentRequest) (*dto.SendDocumentResponse, error)
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
