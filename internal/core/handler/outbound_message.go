package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	corestore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/internal/facebook"
	"github.com/webitel/im-providers-service/internal/provider"
)

// Ensure OutboundMessageHandler implements the generated gRPC server interface.
var _ impb.ProviderMessageServiceServer = (*OutboundMessageHandler)(nil)

// OutboundMessageHandler handles incoming gRPC requests for sending messages.
type OutboundMessageHandler struct {
	logger    *slog.Logger
	registry  *provider.Registry
	store     corestore.GateStore
	typeCache *lru.Cache[string, sharedmodel.GateType]
	impb.UnimplementedProviderMessageServiceServer
}

// NewOutboundMessageHandler creates a new instance of the message handler.
func NewOutboundMessageHandler(logger *slog.Logger, registry *provider.Registry, store corestore.GateStore) *OutboundMessageHandler {
	cache, _ := lru.New[string, sharedmodel.GateType](1000)
	return &OutboundMessageHandler{
		logger:    logger,
		registry:  registry,
		store:     store,
		typeCache: cache,
	}
}

func (p *OutboundMessageHandler) resolveSender(ctx context.Context, gateID string) (provider.Sender, error) {
	var gateType sharedmodel.GateType

	if v, ok := p.typeCache.Get(gateID); ok {
		gateType = v
	} else {
		t, err := p.store.GetTypeByID(ctx, gateID)
		if err != nil {
			if errors.Is(err, corestore.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "gate not found: %s", gateID)
			}
			return nil, status.Errorf(codes.Internal, "failed to resolve gate type for: %s", gateID)
		}
		p.typeCache.Add(gateID, t)
		gateType = t
	}

	if gateType == sharedmodel.TypeUnknown {
		return nil, status.Errorf(codes.InvalidArgument, "unknown gate type for gate: %s", gateID)
	}

	key := gateType.String()
	prov, err := p.registry.Get(key)
	if err != nil {
		return nil, status.Errorf(codes.Unimplemented, "provider not registered: %s", key)
	}
	return prov, nil
}

// SendText handles outgoing plain text messages.
func (p *OutboundMessageHandler) SendText(ctx context.Context, req *impb.ProviderSendTextRequest) (*impb.ProviderSendMessageResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendText"),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
	)
	log.InfoContext(ctx, "outbound text message request received")

	sender, err := p.resolveSender(ctx, req.GetGateId())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID:   req.GetGateId(),
		To:       sharedmodel.Peer{Sub: req.GetExternalUserId()},
		Text:     req.GetText(),
		DomainID: int64(req.DomainId),
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
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
		slog.Int("images_count", len(req.GetImages())),
	)
	log.InfoContext(ctx, "outbound image message request received")

	sender, err := p.resolveSender(ctx, req.GetGateId())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID:   req.GetGateId(),
		To:       sharedmodel.Peer{Sub: req.GetExternalUserId()},
		DomainID: int64(req.DomainId),
		Text:     req.GetCaption(),
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
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
		slog.Int("documents_count", len(req.GetDocuments())),
	)
	log.InfoContext(ctx, "outbound document message request received")

	sender, err := p.resolveSender(ctx, req.GetGateId())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID:   req.GetGateId(),
		To:       sharedmodel.Peer{Sub: req.GetExternalUserId()},
		DomainID: int64(req.DomainId),
		Text:     req.GetCaption(),
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

// SendInteractive handles outgoing interactive messages (buttons, menus).
func (p *OutboundMessageHandler) SendInteractive(ctx context.Context, req *impb.ProviderSendInteractiveRequest) (*impb.ProviderSendMessageResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendInteractive"),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
	)
	log.InfoContext(ctx, "outbound interactive message request received")

	sender, err := p.resolveSender(ctx, req.GetGateId())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
		return nil, err
	}

	is, ok := sender.(provider.InteractiveSender)
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "provider %s does not support interactive messages", sender.Type())
	}

	msg := &sharedmodel.Message{
		GateID:      req.GetGateId(),
		To:          sharedmodel.Peer{Sub: req.GetExternalUserId()},
		Text:        req.GetBody(),
		DomainID:    int64(req.GetDomainId()),
		Interactive: mapInteractive(req.GetInteractive()),
	}

	resp, err := is.SendInteractive(ctx, msg)
	if err != nil {
		log.ErrorContext(ctx, "failed to send interactive message", slog.String("error", err.Error()))
		return nil, toGRPCError(err)
	}

	log.InfoContext(ctx, "interactive message sent", slog.String("external_id", resp.ID))
	return &impb.ProviderSendMessageResponse{
		ExternalId: resp.ID,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

func mapInteractive(pb *impb.ProviderInteractive) *sharedmodel.Interactive {
	if pb == nil {
		return nil
	}
	out := &sharedmodel.Interactive{SingleUse: pb.GetSingleUse()}
	if m := pb.GetMarkup(); m != nil {
		out.Markup = mapMarkup(m)
	} else if l := pb.GetListReply(); l != nil {
		out.ListReply = mapListReply(l)
	}
	return out
}

func mapMarkup(pb *impb.ProviderKeyboardMarkup) *sharedmodel.KeyboardMarkup {
	rows := make([]sharedmodel.KeyboardRow, 0, len(pb.GetRows()))
	for _, r := range pb.GetRows() {
		rows = append(rows, sharedmodel.KeyboardRow{Buttons: mapButtons(r.GetButtons())})
	}
	return &sharedmodel.KeyboardMarkup{Rows: rows}
}

func mapListReply(pb *impb.ProviderKeyboardListReply) *sharedmodel.KeyboardListReply {
	sections := make([]sharedmodel.KeyboardRowWithSection, 0, len(pb.GetSections()))
	for _, s := range pb.GetSections() {
		sections = append(sections, sharedmodel.KeyboardRowWithSection{
			Section: s.GetSection(),
			Buttons: mapButtons(s.GetButtons()),
		})
	}
	return &sharedmodel.KeyboardListReply{
		MainButtonTitle: pb.GetMainButtonTitle(),
		Sections:        sections,
	}
}

func mapButtons(pbs []*impb.ProviderKeyboardButton) []sharedmodel.KeyboardButton {
	out := make([]sharedmodel.KeyboardButton, 0, len(pbs))
	for _, b := range pbs {
		btn := sharedmodel.KeyboardButton{ID: b.GetId(), Label: b.GetLabel()}
		switch {
		case b.GetUrl() != nil:
			btn.URL = &sharedmodel.KeyboardButtonURL{URL: b.GetUrl().GetUrl()}
		case b.GetCallback() != nil:
			btn.Callback = &sharedmodel.KeyboardButtonCallback{Data: b.GetCallback().GetData()}
		case b.GetRequest() != nil:
			btn.Request = &sharedmodel.KeyboardButtonRequest{Action: b.GetRequest().GetAction()}
		}
		out = append(out, btn)
	}
	return out
}

func toGRPCError(err error) error {
	if errors.Is(err, facebook.ErrTokenInvalid) {
		return status.Errorf(codes.Unauthenticated, "page token invalid or revoked: re-authorize via StartMetaOAuth")
	}
	return err
}
