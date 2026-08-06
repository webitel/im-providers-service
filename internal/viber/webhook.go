package viber

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/metadata"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

// inboundEvent is the top-level Viber callback envelope.
// https://developers.viber.com/docs/api/rest-bot-api/#callbacks
type inboundEvent struct {
	Event        string          `json:"event"`
	MessageToken json.Number     `json:"message_token"`
	Sender       *inboundSender  `json:"sender"`
	Message      *inboundMessage `json:"message"`
	UserID       string          `json:"user_id"`
	Desc         string          `json:"desc"`
	User         *inboundSender  `json:"user"`
	Subscribed   bool            `json:"subscribed"`
}

func (e *inboundEvent) actor() *inboundSender {
	if e.Sender != nil && e.Sender.ID != "" {
		return e.Sender
	}
	if e.User != nil && e.User.ID != "" {
		return e.User
	}
	if e.UserID != "" {
		return &inboundSender{ID: e.UserID}
	}

	return nil
}

type inboundSender struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type inboundMessage struct {
	Type         string           `json:"type"`
	Text         string           `json:"text"`
	Media        string           `json:"media"`
	Size         int64            `json:"size"`
	FileSize     int64            `json:"file_size"`
	Duration     int              `json:"duration"`
	FileName     string           `json:"file_name"`
	TrackingData string           `json:"tracking_data"`
	StickerID    int64            `json:"sticker_id"`
	Location     *inboundLocation `json:"location"`
	Contact      *inboundContact  `json:"contact"`
}

func (m *inboundMessage) byteSize() int64 {
	if m.Size > 0 {
		return m.Size
	}
	return m.FileSize
}

type inboundLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type inboundContact struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
}

func (p *viberProvider) HandleWebhook(ctx context.Context, data []byte) error {
	var evt inboundEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		p.logger.Warn("failed to parse viber webhook", "err", err)
		return nil
	}

	switch evt.Event {
	case "message":
		return p.handleMessage(ctx, &evt)
	case "conversation_started":
		return p.handleConversationStarted(ctx, &evt)
	case "subscribed":
		return p.handleSubscription(ctx, &evt, true)
	case "unsubscribed":
		return p.handleSubscription(ctx, &evt, false)
	case "delivered", "seen":
		return p.handleDeliveryStatus(ctx, &evt)
	case "failed":
		return p.handleFailed(ctx, &evt)
	default:
		p.logger.Debug("viber event ignored", "event", evt.Event)
		return nil
	}
}

func (p *viberProvider) handleConversationStarted(ctx context.Context, evt *inboundEvent) error {
	actor := evt.actor()
	if actor == nil {
		return nil
	}

	gate, err := p.gateFor(ctx)
	if err != nil || gate == nil {
		return err
	}

	if _, err := p.syncContact(ctx, gate, actor); err != nil {
		return fmt.Errorf("sync contact [id=%s]: %w", actor.ID, err)
	}

	p.setSubscription(ctx, gate.ID, actor.ID, evt.Subscribed)
	p.openWelcomeWindow(ctx, gate.ID, actor.ID)
	p.sendWelcome(ctx, gate, actor)

	return nil
}

func (p *viberProvider) sendWelcome(ctx context.Context, gate *vibmodel.ViberGate, actor *inboundSender) {
	if p.templates == nil {
		return
	}

	body := p.templates.Render(ctx, gate.ID, sharedsvc.EventConversationStarted, map[string]string{
		"client_name": actor.Name,
		"gate_name":   gate.Name,
	})
	if body == "" {
		return
	}

	if _, err := p.api.SendText(ctx, gate.AuthToken, senderOf(gate), actor.ID, body, nil); err != nil {
		p.logger.WarnContext(ctx, "viber welcome message failed", "gate_id", gate.ID, "err", err)
	}
}

func (p *viberProvider) handleSubscription(ctx context.Context, evt *inboundEvent, subscribed bool) error {
	actor := evt.actor()
	if actor == nil {
		return nil
	}

	gate, err := p.gateFor(ctx)
	if err != nil || gate == nil {
		return err
	}

	p.setSubscription(ctx, gate.ID, actor.ID, subscribed)

	if subscribed {
		if _, err := p.syncContact(ctx, gate, actor); err != nil {
			return fmt.Errorf("sync contact [id=%s]: %w", actor.ID, err)
		}

		return nil
	}

	p.logger.InfoContext(ctx, "viber user unsubscribed", "gate_id", gate.ID, "user", actor.ID)

	return nil
}

func (p *viberProvider) handleFailed(ctx context.Context, evt *inboundEvent) error {
	p.logger.WarnContext(ctx, "viber delivery failed",
		"token", evt.MessageToken.String(),
		"user", evt.UserID,
		"desc", evt.Desc,
	)

	return p.reportDelivery(ctx, evt, deliveryFailed)
}

func (p *viberProvider) handleDeliveryStatus(ctx context.Context, evt *inboundEvent) error {
	status := deliveryDelivered
	if evt.Event == "seen" {
		status = deliveryRead
	}

	return p.reportDelivery(ctx, evt, status)
}

func (p *viberProvider) gateFor(ctx context.Context) (*vibmodel.ViberGate, error) {
	uri := p.webhookURI(ctx)

	gate, err := p.resolveGate(ctx, uri)
	if err != nil {
		return nil, err
	}
	if gate == nil || !gate.Enabled {
		p.logger.WarnContext(ctx, "viber webhook for a missing or disabled gate, dropping")

		return nil, nil
	}

	return gate, nil
}

func (p *viberProvider) handleMessage(ctx context.Context, evt *inboundEvent) error {
	if evt.Sender == nil || evt.Sender.ID == "" || evt.Message == nil {
		return nil
	}

	gate, err := p.gateFor(ctx)
	if err != nil || gate == nil {
		return err
	}

	token := evt.MessageToken.String()
	if token != "" && messageSeen(ctx, p.rdb, token) {
		p.logger.DebugContext(ctx, "duplicate viber event skipped", "token", token)
		return nil
	}

	if _, err := p.syncContact(ctx, gate, evt.Sender); err != nil {
		return fmt.Errorf("sync contact [id=%s]: %w", evt.Sender.ID, err)
	}

	peers := peerPair{
		from: sharedmodel.Peer{Sub: evt.Sender.ID, Iss: gate.Peer.Iss, Name: evt.Sender.Name},
		to:   sharedmodel.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss},
	}

	sendCtx := withProviderVia(ctx, gate.DomainID, peers.from.Sub, gate.ID)
	p.routeMessage(sendCtx, gate, peers, evt.Message, token)
	return nil
}

func withProviderVia(ctx context.Context, dc int64, customerSub, gateID string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-webitel-type", "provider",
		"x-webitel-provider", fmt.Sprintf("%d.%s", dc, customerSub),
		"x-webitel-via", gateID,
	))
}
