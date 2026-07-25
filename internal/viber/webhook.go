package viber

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/metadata"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
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
}

type inboundSender struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type inboundMessage struct {
	Type     string           `json:"type"`
	Text     string           `json:"text"`
	Media    string           `json:"media"`
	Size     int64            `json:"size"`
	FileName string           `json:"file_name"`
	Location *inboundLocation `json:"location"`
	Contact  *inboundContact  `json:"contact"`
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
	case "failed":
		p.logger.Warn("viber delivery failed", "token", evt.MessageToken.String(), "user", evt.UserID, "desc", evt.Desc)
		return nil
	default:
		p.logger.Debug("viber event ignored", "event", evt.Event)
		return nil
	}
}

// handleMessage runs the per-message pipeline: resolve gate → dedup → sync contact → route.
func (p *viberProvider) handleMessage(ctx context.Context, evt *inboundEvent) error {
	if evt.Sender == nil || evt.Sender.ID == "" || evt.Message == nil {
		return nil
	}

	gate, err := p.resolveGate(ctx, p.webhookURI(ctx))
	if err != nil || !gate.Enabled {
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

// withProviderVia stamps the provider caller identity (the customer) and the gate id
// onto the outgoing gRPC metadata used for the inbound message forward.
func withProviderVia(ctx context.Context, dc int64, customerSub, gateID string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"x-webitel-type", "provider",
		"x-webitel-provider", fmt.Sprintf("%d.%s", dc, customerSub),
		"x-webitel-via", gateID,
	))
}
