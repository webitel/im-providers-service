package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/provider/facebook"
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

func (p *ProviderMessageHandler) resolveSender(t impb.ProviderType) (provider.Sender, error) {
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
	s, ok := p.senders[key]
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "provider not registered: %s", key)
	}
	return s, nil
}

// SendText handles outgoing plain text messages.
func (p *ProviderMessageHandler) SendText(ctx context.Context, req *impb.ProviderSendTextRequest) (*impb.ProviderSendMessageResponse, error) {
	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		return nil, err
	}

	msg := &model.Message{
		GateID: req.GetGateId(),
		To:     model.Peer{Sub: req.GetExternalUserId()},
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
func (p *ProviderMessageHandler) SendImage(ctx context.Context, req *impb.ProviderSendImageRequest) (*impb.ProviderSendMessageResponse, error) {
	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		return nil, err
	}

	msg := &model.Message{
		GateID: req.GetGateId(),
		To:     model.Peer{Sub: req.GetExternalUserId()},
	}
	if f := req.GetImage(); f != nil {
		msg.Images = []*model.Image{{
			ID:       f.GetId(),
			URL:      f.GetUrl(),
			FileName: f.GetName(),
			MimeType: f.GetMimeType(),
			Size:     f.GetSize(),
		}}
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
func (p *ProviderMessageHandler) SendDocument(ctx context.Context, req *impb.ProviderSendDocumentRequest) (*impb.ProviderSendMessageResponse, error) {
	sender, err := p.resolveSender(req.GetType())
	if err != nil {
		return nil, err
	}

	msg := &model.Message{
		GateID: req.GetGateId(),
		To:     model.Peer{Sub: req.GetExternalUserId()},
	}
	if f := req.GetDocument(); f != nil {
		msg.Documents = []*model.Document{{
			ID:       f.GetId(),
			URL:      f.GetUrl(),
			FileName: f.GetName(),
			MimeType: f.GetMimeType(),
			Size:     f.GetSize(),
		}}
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
