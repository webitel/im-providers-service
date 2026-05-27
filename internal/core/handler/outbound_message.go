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
		p.logger.Debug("routing to facebook sender")
	case impb.ProviderType_PROVIDER_TYPE_WHATSAPP:
		key = "whatsapp"
		p.logger.Debug("routing to whatsapp sender")
	case impb.ProviderType_PROVIDER_TYPE_INSTAGRAM:
		p.logger.Warn("provider not implemented", slog.String("provider", "instagram"))
		return nil, status.Errorf(codes.Unimplemented, "instagram provider not implemented yet")
	case impb.ProviderType_PROVIDER_TYPE_TELEGRAM_APP:
		p.logger.Warn("provider not implemented", slog.String("provider", "telegram_app"))
		return nil, status.Errorf(codes.Unimplemented, "telegram app provider not implemented yet")
	case impb.ProviderType_PROVIDER_TYPE_TELEGRAM_BOT:
		p.logger.Warn("provider not implemented", slog.String("provider", "telegram_bot"))
		return nil, status.Errorf(codes.Unimplemented, "telegram bot provider not implemented yet")
	case impb.ProviderType_PROVIDER_TYPE_VIBER:
		p.logger.Warn("provider not implemented", slog.String("provider", "viber"))
		return nil, status.Errorf(codes.Unimplemented, "viber provider not implemented yet")
	default:
		p.logger.Warn("unsupported provider type", slog.String("provider", t.String()))
		return nil, status.Errorf(codes.InvalidArgument, "unsupported provider type: %s", t)
	}
	p.logger.Debug("resolved sender", slog.String("provider", key))
	prov, err := p.registry.Get(key)
	if err != nil {
		p.logger.Error("provider not registered in registry", slog.String("provider", key))
		return nil, status.Errorf(codes.Unimplemented, "provider not registered: %s", key)
	}
	return prov, nil
}

// SendText handles outgoing plain text messages.
func (p *OutboundMessageHandler) SendText(ctx context.Context, req *impb.ProviderSendTextRequest) (*impb.ProviderSendMessageResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendText"),
		slog.String("provider", req.GetType().String()),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
	)
	log.InfoContext(ctx, "outbound text message request received")

	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID: req.GetGateId(),
		To:     sharedmodel.Peer{Sub: req.GetExternalUserId()},
		Text:   req.GetText(),
	}

	resp, err := sender.SendText(ctx, msg)
	if err != nil {
		log.ErrorContext(ctx, "failed to send text message", slog.String("error", err.Error()))
		return nil, toGRPCError(err)
	}

	log.InfoContext(ctx, "text message sent", slog.String("external_id", resp.ID))
	return &impb.ProviderSendMessageResponse{
		ExternalId: resp.ID,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

// SendImage handles outgoing messages containing images.
func (p *OutboundMessageHandler) SendImage(ctx context.Context, req *impb.ProviderSendImageRequest) (*impb.ProviderSendMessageResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendImage"),
		slog.String("provider", req.GetType().String()),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
		slog.Int("images_count", len(req.GetImages())),
	)
	log.InfoContext(ctx, "outbound image message request received")

	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
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
		log.ErrorContext(ctx, "failed to send image message", slog.String("error", err.Error()))
		return nil, toGRPCError(err)
	}

	log.InfoContext(ctx, "image message sent", slog.String("external_id", resp.ID))
	return &impb.ProviderSendMessageResponse{
		ExternalId: resp.ID,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

// SendDocument handles outgoing messages containing documents/files.
func (p *OutboundMessageHandler) SendDocument(ctx context.Context, req *impb.ProviderSendDocumentRequest) (*impb.ProviderSendMessageResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendDocument"),
		slog.String("provider", req.GetType().String()),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
		slog.Int("documents_count", len(req.GetDocuments())),
	)
	log.InfoContext(ctx, "outbound document message request received")

	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
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
		log.ErrorContext(ctx, "failed to send document message", slog.String("error", err.Error()))
		return nil, toGRPCError(err)
	}

	log.InfoContext(ctx, "document message sent", slog.String("external_id", resp.ID))
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
