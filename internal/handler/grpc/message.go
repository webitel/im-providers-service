package grpc

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/internal/provider"
)

// Ensure ProviderMessageHandler implements the generated gRPC server interface.
var _ impb.ProviderMessageServiceServer = (*ProviderMessageHandler)(nil)

// ProviderMessageHandler handles incoming gRPC requests for sending messages.
type ProviderMessageHandler struct {
	logger *slog.Logger
	// Registry of platform-specific senders (Facebook, WhatsApp, etc.)
	senders map[string]provider.Sender
	impb.UnimplementedProviderMessageServiceServer
}

// NewProviderMessageHandler creates a new instance of the message handler.
func NewProviderMessageHandler(logger *slog.Logger, senders []provider.Sender) *ProviderMessageHandler {
	m := make(map[string]provider.Sender)
	for _, s := range senders {
		m[s.Type()] = s
	}
	return &ProviderMessageHandler{
		logger:  logger,
		senders: m,
	}
}

// SendText handles outgoing plain text messages.
func (p *ProviderMessageHandler) SendText(ctx context.Context, req *impb.ProviderSendTextRequest) (*impb.ProviderSendMessageResponse, error) {
	p.logger.Info("received SendText request", "gate_id", req.GetGateId(), "type", req.GetType())

	// TODO: Resolve sender by req.Type and call sender.SendText
	// For now, returning a stub response
	return &impb.ProviderSendMessageResponse{
		ExternalId: "stub_text_id",
		CreatedAt:  1712589535, // Example timestamp
	}, nil
}

// SendImage handles outgoing messages containing images.
func (p *ProviderMessageHandler) SendImage(ctx context.Context, req *impb.ProviderSendImageRequest) (*impb.ProviderSendMessageResponse, error) {
	p.logger.Info("received SendImage request", "gate_id", req.GetGateId())

	// TODO: Map impb.ProviderFile to model.Image and call sender.SendImage
	return &impb.ProviderSendMessageResponse{
		ExternalId: "stub_image_id",
		CreatedAt:  1712589535,
	}, nil
}

// SendDocument handles outgoing messages containing documents/files.
func (p *ProviderMessageHandler) SendDocument(ctx context.Context, req *impb.ProviderSendDocumentRequest) (*impb.ProviderSendMessageResponse, error) {
	p.logger.Info("received SendDocument request", "gate_id", req.GetGateId())

	// TODO: Map impb.ProviderFile to model.Document and call sender.SendDocument
	return &impb.ProviderSendMessageResponse{
		ExternalId: "stub_doc_id",
		CreatedAt:  1712589535,
	}, nil
}
