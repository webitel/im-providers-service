package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/webitel-go-kit/pkg/cache"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	coreservice "github.com/webitel/im-providers-service/internal/core/service"
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
	typeCache cache.Cache[string, sharedmodel.GateType]
	templates *coreservice.TemplateRenderer
	status    *coreservice.StatusReporter
	impb.UnimplementedProviderMessageServiceServer
}

// NewOutboundMessageHandler creates a new instance of the message handler.
func NewOutboundMessageHandler(
	logger *slog.Logger,
	registry *provider.Registry,
	store corestore.GateStore,
	templates *coreservice.TemplateRenderer,
	status *coreservice.StatusReporter,
) (*OutboundMessageHandler, error) {
	typeCache, err := cache.New[string, sharedmodel.GateType]().
		L1(cache.RistrettoConfig{MaxCost: 1000, NumCounters: 10000}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("outbound handler: init type cache: %w", err)
	}
	return &OutboundMessageHandler{
		logger:    logger,
		registry:  registry,
		store:     store,
		typeCache: typeCache,
		templates: templates,
		status:    status,
	}, nil
}

func (p *OutboundMessageHandler) resolveSender(ctx context.Context, gateID string) (provider.Sender, error) {
	var gateType sharedmodel.GateType

	if v, ok, _ := p.typeCache.Get(ctx, gateID); ok {
		gateType = v
	} else {
		t, err := p.store.GetTypeByID(ctx, gateID)
		if err != nil {
			if errors.Is(err, corestore.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "gate not found: %s", gateID)
			}
			return nil, status.Errorf(codes.Internal, "failed to resolve gate type for: %s", gateID)
		}
		_ = p.typeCache.Set(ctx, gateID, t)
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

	mc := messageContextOf(req.GetGateId(), req.GetExternalUserId(), req.GetMessageId(), req.GetThreadId(), req.GetDomainId())

	externalContactID, err := uuid.Parse(req.GetExternalUserId())
	if err != nil {
		return nil, err
	}

	msg := &sharedmodel.Message{
		GateID: req.GetGateId(),
		// TODO: remove the sub
		//  req.GetExternalUserId() is the CONTACT ID!!! not CONTACT SUB!!!
		// I added ID to the peer to avoid confusion and saved the SUB for the backward compatibility
		// for the providers
		To:                sharedmodel.Peer{ID: externalContactID, Sub: req.GetExternalUserId()},
		Text:              req.GetText(),
		DomainID:          int64(req.DomainId),
		ReplyToExternalID: req.GetReplyToExternalId(),
	}

	resp, err := sender.SendText(ctx, msg)
	p.trackOutcome(ctx, mc, resp, err)
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
		GateID:            req.GetGateId(),
		To:                sharedmodel.Peer{Sub: req.GetExternalUserId()},
		DomainID:          int64(req.DomainId),
		Text:              req.GetCaption(),
		ReplyToExternalID: req.GetReplyToExternalId(),
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
	p.trackOutcome(ctx, messageContextOf(req.GetGateId(), req.GetExternalUserId(), req.GetMessageId(), req.GetThreadId(), req.GetDomainId()), resp, err)
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
		GateID:            req.GetGateId(),
		To:                sharedmodel.Peer{Sub: req.GetExternalUserId()},
		DomainID:          int64(req.DomainId),
		Text:              req.GetCaption(),
		ReplyToExternalID: req.GetReplyToExternalId(),
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
	p.trackOutcome(ctx, messageContextOf(req.GetGateId(), req.GetExternalUserId(), req.GetMessageId(), req.GetThreadId(), req.GetDomainId()), resp, err)
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
		GateID:            req.GetGateId(),
		To:                sharedmodel.Peer{Sub: req.GetExternalUserId()},
		Text:              req.GetBody(),
		DomainID:          int64(req.GetDomainId()),
		Interactive:       mapInteractive(req.GetInteractive()),
		ReplyToExternalID: req.GetReplyToExternalId(),
	}

	resp, err := is.SendInteractive(ctx, msg)
	p.trackOutcome(ctx, messageContextOf(req.GetGateId(), req.GetExternalUserId(), req.GetMessageId(), req.GetThreadId(), req.GetDomainId()), resp, err)
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

// SendSystemMessage renders a system event template and delivers the result as a
// plain text message to the external chat partner via the appropriate provider.
// If the rendered text is empty (e.g. the gate has a blank template override),
// the call is a no-op and returns success so the caller is not penalised.
func (p *OutboundMessageHandler) SendSystemMessage(ctx context.Context, req *impb.ProviderSendSystemMessageRequest) (*impb.ProviderSendMessageResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendSystemMessage"),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
		slog.String("event_type", req.GetEventType()),
	)
	log.InfoContext(ctx, "outbound system message request received")

	sender, err := p.resolveSender(ctx, req.GetGateId())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
		return nil, err
	}

	text := p.templates.Render(ctx, req.GetGateId(), req.GetEventType(), req.GetVars())
	if text == "" {
		return &impb.ProviderSendMessageResponse{CreatedAt: time.Now().Unix()}, nil
	}

	msg := &sharedmodel.Message{
		GateID:   req.GetGateId(),
		To:       sharedmodel.Peer{Sub: req.GetExternalUserId()},
		Text:     text,
		DomainID: int64(req.GetDomainId()),
	}

	resp, err := sender.SendText(ctx, msg)
	p.trackOutcome(ctx, messageContextOf(req.GetGateId(), req.GetExternalUserId(), req.GetMessageId(), req.GetThreadId(), req.GetDomainId()), resp, err)
	if err != nil {
		log.ErrorContext(ctx, "failed to send system message", slog.String("error", err.Error()))
		return nil, toGRPCError(err)
	}

	log.InfoContext(ctx, "system message sent", slog.String("external_id", resp.ID))
	return &impb.ProviderSendMessageResponse{
		ExternalId: resp.ID,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

// SendTyping forwards an ephemeral typing indicator to the external chat
// partner. Best-effort: channels whose provider does not implement TypingSender
// are a silent no-op (success), so callers need not special-case them.
func (p *OutboundMessageHandler) SendTyping(ctx context.Context, req *impb.ProviderSendTypingRequest) (*impb.ProviderSendTypingResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendTyping"),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
		slog.Bool("typing_on", req.GetTypingOn()),
	)

	sender, err := p.resolveSender(ctx, req.GetGateId())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))
		return nil, err
	}

	ts, ok := sender.(provider.TypingSender)
	if !ok {
		// Channel does not support typing — no-op success.
		log.DebugContext(ctx, "provider does not support typing, skipping", slog.String("type", sender.Type()))
		return &impb.ProviderSendTypingResponse{}, nil
	}

	if err := ts.SendTyping(ctx, &provider.TypingRequest{
		GateID:     req.GetGateId(),
		ExternalID: req.GetExternalUserId(),
		DomainID:   req.GetDomainId(),
		TypingOn:   req.GetTypingOn(),
	}); err != nil {
		log.WarnContext(ctx, "failed to send typing", slog.String("error", err.Error()))
		return nil, toGRPCError(err)
	}

	return &impb.ProviderSendTypingResponse{}, nil
}

// SendReaction forwards an emoji reaction change to the external chat partner.
// Best-effort: channels whose provider does not implement ReactionSender
// are a silent no-op (success), so callers need not special-case them.
func (p *OutboundMessageHandler) SendReaction(ctx context.Context, req *impb.ProviderSendReactionRequest) (*impb.ProviderSendReactionResponse, error) {
	log := p.logger.With(
		slog.String("method", "SendReaction"),
		slog.String("gate_id", req.GetGateId()),
		slog.String("external_user_id", req.GetExternalUserId()),
		slog.String("emoji", req.GetEmoji()),
	)

	sender, err := p.resolveSender(ctx, req.GetGateId())
	if err != nil {
		log.WarnContext(ctx, "failed to resolve sender", slog.String("error", err.Error()))

		return nil, err
	}

	rs, ok := sender.(provider.ReactionSender)
	if !ok {
		// Channel does not support reactions — no-op success.
		log.DebugContext(ctx, "provider does not support reactions, skipping", slog.String("type", sender.Type()))

		return &impb.ProviderSendReactionResponse{}, nil
	}

	if err := rs.SendReaction(ctx, &provider.ReactionRequest{
		GateID:            req.GetGateId(),
		ExternalID:        req.GetExternalUserId(),
		ExternalMessageID: req.GetExternalMessageId(),
		DomainID:          req.GetDomainId(),
		Emoji:             req.GetEmoji(),
		Removed:           req.GetRemoved(),
	}); err != nil {
		log.WarnContext(ctx, "failed to send reaction", slog.String("error", err.Error()))

		return nil, toGRPCError(err)
	}

	return &impb.ProviderSendReactionResponse{}, nil
}

// messageContext is the internal message identity carried by outbound
// requests for delivery status tracking.
type messageContext struct {
	gateID         string
	externalUserID string
	messageID      string
	threadID       string
	domainID       int32
}

func messageContextOf(gateID, externalUserID, messageID, threadID string, domainID int32) messageContext {
	return messageContext{
		gateID:         gateID,
		externalUserID: externalUserID,
		messageID:      messageID,
		threadID:       threadID,
		domainID:       domainID,
	}
}

// trackOutcome persists the provider message ref on success and reports a
// FAILED status on synchronous send errors. Requests without message context
// (older callers) are not tracked.
func (p *OutboundMessageHandler) trackOutcome(ctx context.Context, mc messageContext, resp *sharedmodel.MessageResponse, sendErr error) {
	if mc.messageID == "" || mc.threadID == "" {
		return
	}

	if sendErr != nil {
		p.status.SendFailure(ctx, coreservice.SendFailureReport{
			MessageID:    mc.messageID,
			ThreadID:     mc.threadID,
			MemberID:     mc.externalUserID,
			DomainID:     mc.domainID,
			ErrorCode:    status.Code(sendErr).String(),
			ErrorMessage: sendErr.Error(),
		})

		return
	}

	if resp == nil || resp.ID == "" {
		return
	}

	messageID, err := uuid.Parse(mc.messageID)
	if err != nil {
		return
	}

	threadID, err := uuid.Parse(mc.threadID)
	if err != nil {
		return
	}

	memberID, err := uuid.Parse(mc.externalUserID)
	if err != nil {
		return
	}

	providerUserID := ""
	if v, ok := resp.MD["recipient_id"].(string); ok {
		providerUserID = v
	}

	p.status.SaveRef(ctx, &sharedmodel.MessageRef{
		GateID:            mc.gateID,
		ProviderMessageID: resp.ID,
		ProviderUserID:    providerUserID,
		MessageID:         messageID,
		ThreadID:          threadID,
		MemberID:          memberID,
		DomainID:          int64(mc.domainID),
	})
}

func toGRPCError(err error) error {
	if errors.Is(err, facebook.ErrTokenInvalid) {
		return status.Errorf(codes.Unauthenticated, "page token invalid or revoked: re-authorize via StartMetaOAuth")
	}
	return err
}
