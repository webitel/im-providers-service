package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/facebook"
	"github.com/webitel/im-providers-service/internal/provider"
)

// Ensure OutboundMessageHandler implements the generated gRPC server interface.
var _ impb.ProviderMessageServiceServer = (*OutboundMessageHandler)(nil)

// OutboundMessageHandler handles incoming gRPC requests for sending messages.
type OutboundMessageHandler struct {
	logger   *slog.Logger
	registry *provider.Registry
	impb.UnimplementedProviderMessageServiceServer
}

// NewOutboundMessageHandler creates a new instance of the message handler.
func NewOutboundMessageHandler(logger *slog.Logger, registry *provider.Registry) *OutboundMessageHandler {
	return &OutboundMessageHandler{
		logger:   logger,
		registry: registry,
	}
}

func (p *OutboundMessageHandler) resolveSender(t impb.ProviderType) (provider.Sender, error) {
	var key string
	switch t {
	case impb.ProviderType_PROVIDER_TYPE_FACEBOOK:
		key = "facebook"
	case impb.ProviderType_PROVIDER_TYPE_WHATSAPP:
		key = "whatsapp"
	case impb.ProviderType_PROVIDER_TYPE_INSTAGRAM:
		return nil, status.Errorf(codes.Unimplemented, "instagram provider not implemented yet")
	case impb.ProviderType_PROVIDER_TYPE_TELEGRAM_APP:
		return nil, status.Errorf(codes.Unimplemented, "telegram app provider not implemented yet")
	case impb.ProviderType_PROVIDER_TYPE_TELEGRAM_BOT:
		return nil, status.Errorf(codes.Unimplemented, "telegram bot provider not implemented yet")
	case impb.ProviderType_PROVIDER_TYPE_VIBER:
		return nil, status.Errorf(codes.Unimplemented, "viber provider not implemented yet")
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported provider type: %s", t)
	}
	prov, err := p.registry.Get(key)
	if err != nil {
		return nil, status.Errorf(codes.Unimplemented, "provider not registered: %s", key)
	}
	return prov, nil
}

// SendText handles outgoing plain text messages.
func (p *OutboundMessageHandler) SendText(ctx context.Context, req *impb.ProviderSendTextRequest) (*impb.ProviderSendMessageResponse, error) {
	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID: req.GetGateId(),
		To:     sharedmodel.Peer{Sub: req.GetExternalUserId()},
		Text:   req.GetText(),
	}

	resp, err := sender.SendText(ctx, msg)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &impb.ProviderSendMessageResponse{
		ExternalId: resp.ID,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

// SendImage handles outgoing messages containing images.
func (p *OutboundMessageHandler) SendImage(ctx context.Context, req *impb.ProviderSendImageRequest) (*impb.ProviderSendMessageResponse, error) {
	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID: req.GetGateId(),
		To:     sharedmodel.Peer{Sub: req.GetExternalUserId()},
	}
	for _, f := range req.GetImages() {
		msg.Images = append(msg.Images, &sharedmodel.Image{
			ID:       f.GetId(),
			URL:      f.GetUrl(),
			FileName: f.GetName(),
			MimeType: f.GetMimeType(),
			Size:     f.GetSize(),
		})
	}

	resp, err := sender.SendImage(ctx, msg)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &impb.ProviderSendMessageResponse{
		ExternalId: resp.ID,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

// SendDocument handles outgoing messages containing documents/files.
func (p *OutboundMessageHandler) SendDocument(ctx context.Context, req *impb.ProviderSendDocumentRequest) (*impb.ProviderSendMessageResponse, error) {
	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID: req.GetGateId(),
		To:     sharedmodel.Peer{Sub: req.GetExternalUserId()},
	}
	for _, f := range req.GetDocuments() {
		msg.Documents = append(msg.Documents, &sharedmodel.Document{
			ID:       f.GetId(),
			URL:      f.GetUrl(),
			FileName: f.GetName(),
			MimeType: f.GetMimeType(),
			Size:     f.GetSize(),
		})
	}

	resp, err := sender.SendDocument(ctx, msg)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &impb.ProviderSendMessageResponse{
		ExternalId: resp.ID,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

func toGRPCError(err error) error {
	if errors.Is(err, facebook.ErrTokenInvalid) {
		return status.Errorf(codes.Unauthenticated, "page token invalid or revoked: re-authorize via StartMetaOAuth")
	}
	return err
}
