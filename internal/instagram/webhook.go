package instagram

import (
	"context"
	"fmt"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

func (p *instagramProvider) HandleWebhook(ctx context.Context, data []byte) error {
	evt, err := p.api.ParseWebhook(data)
	if err != nil || evt == nil {
		return nil
	}

	uri := p.webhookURI(ctx)

	// Each entry may belong to a different Instagram business account, so resolve
	// the gate per entry — never assume Entry[0] owns every message in the batch.
	for i := range evt.Entry {
		entry := evt.Entry[i]

		gate, err := p.resolveGate(ctx, uri, entry.ID)
		if err != nil {
			p.logger.Error("gate resolve failed", "entry_id", entry.ID, "err", err)

			continue
		}

		if !gate.Enabled {
			continue
		}

		for _, msg := range entry.Messaging {
			if err := p.processMessage(ctx, gate, msg); err != nil {
				p.logger.Error("message dropped", "sender", msg.Sender.ID, "err", err)
			}
		}
	}

	return nil
}

// processMessage is the per-event pipeline:
//
//	dedup → fetch profile → sync contact → route content
func (p *instagramProvider) processMessage(ctx context.Context, gate *igmodel.InstagramGate, msg Messaging) error {
	igsid := msg.Sender.ID
	if igsid == "" {
		return nil
	}

	if msg.Delivery != nil || msg.Read != nil {
		p.processReceipt(ctx, gate, igsid, msg)

		return nil
	}

	if msg.Message == nil && msg.Postback == nil {
		return nil
	}

	if msg.Message != nil && msg.Message.IsEcho {
		return nil
	}

	// Deduplicate: Instagram may deliver the same event more than once.
	mid := messageID(msg)
	if mid != "" && messageSeen(ctx, p.rdb, mid) {
		p.logger.DebugContext(ctx, "duplicate instagram event skipped", "mid", mid)

		return nil
	}

	profile, err := p.api.GetUserProfile(ctx, igsid, gate.IGToken)
	if err != nil {
		return fmt.Errorf("fetch profile [igsid=%s]: %w", igsid, err)
	}

	_, firstContact, err := p.syncContact(ctx, gate, igsid, profile)
	if err != nil {
		return fmt.Errorf("sync contact [igsid=%s]: %w", igsid, err)
	}

	peers := peerPair{
		from: sharedmodel.Peer{Sub: igsid, Iss: gate.Peer.Iss},
		to:   sharedmodel.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss, Via: &gate.ID},
	}

	p.logger.DebugContext(ctx, "instagram inbound peers built",
		"from_sub", peers.from.Sub,
		"from_iss", peers.from.Iss,
		"to_sub", peers.to.Sub,
		"to_iss", peers.to.Iss,
		"to_via", gate.ID,
	)

	if msg.Message != nil {
		p.recordInboundTimestamp(ctx, gate.ID, igsid)
		p.routeMessage(ctx, gate, peers, msg.Message)
	}

	if msg.Postback != nil {
		p.recordInboundTimestamp(ctx, gate.ID, igsid)
		p.routePostback(ctx, gate, peers, msg.Postback)
	}

	// Only greet on the very first contact — not on every inbound message.
	if firstContact {
		p.sendConversationStartedTemplate(ctx, gate, igsid, profile)
	}

	return nil
}

// processReceipt maps Instagram delivery/read receipts to delivery-status
// reports for im-thread-service. Delivery receipts with explicit mids are
// resolved directly; watermark-only receipts confirm every message sent to
// the user before the watermark.
func (p *instagramProvider) processReceipt(ctx context.Context, gate *igmodel.InstagramGate, igsid string, msg Messaging) {
	if d := msg.Delivery; d != nil {
		at := watermarkTime(d.Watermark, msg.Timestamp)

		if len(d.Mids) > 0 {
			p.status.DeliveredByProviderIDs(ctx, gate.ID, d.Mids, at)
		} else {
			p.status.DeliveredUpTo(ctx, gate.ID, igsid, at)
		}
	}

	if r := msg.Read; r != nil {
		p.status.ReadUpTo(ctx, gate.ID, igsid, watermarkTime(r.Watermark, msg.Timestamp))
	}
}

// watermarkTime converts a receipt watermark (Unix ms) to time, falling back
// to the event timestamp and finally to "now".
func watermarkTime(watermark, eventTimestamp int64) time.Time {
	if watermark > 0 {
		return time.UnixMilli(watermark)
	}

	if eventTimestamp > 0 {
		return time.UnixMilli(eventTimestamp)
	}

	return time.Now()
}

// sendConversationStartedTemplate renders and sends a conversation_started template message
// if configured for this gate. Called on first inbound message from a user.
func (p *instagramProvider) sendConversationStartedTemplate(ctx context.Context, gate *igmodel.InstagramGate, igsid string, profile *UserProfile) {
	if p.templates == nil {
		return
	}

	body := p.templates.Render(ctx, gate.ID, sharedsvc.EventConversationStarted, map[string]string{
		"client_name": profile.Name,
		"gate_name":   gate.Name,
	})
	if body == "" {
		return
	}

	if _, err := p.api.SendText(ctx, gate.IGToken, igsid, body); err != nil {
		p.logger.WarnContext(ctx, "instagram conversation started template failed", "gate_id", gate.ID, "err", err)
	}
}

// messageID returns the Instagram message or postback mid for deduplication.
func messageID(msg Messaging) string {
	if msg.Message != nil {
		return msg.Message.Mid
	}

	if msg.Postback != nil {
		return msg.Postback.Mid
	}

	return ""
}

// routeMessage dispatches inbound text and attachment content to the messenger.
// Errors are logged and non-fatal: a single failed delivery must not block others.
func (p *instagramProvider) routeMessage(ctx context.Context, gate *igmodel.InstagramGate, peers peerPair, msg *InboundMessage) {
	replyTo := ""
	if msg.ReplyTo != nil {
		replyTo = msg.ReplyTo.Mid
	}

	if msg.Text != "" {
		if _, err := p.coreMessengerFor(gate).SendText(ctx, &sharedmodel.SendTextRequest{
			DomainID:          gate.DomainID,
			From:              peers.from,
			To:                peers.to,
			Body:              msg.Text,
			ExternalID:        msg.Mid,
			ReplyToExternalID: replyTo,
		}); err != nil {
			p.logger.Error("send text failed", "err", err)
		}
	}

	if len(msg.Attachments) > 0 {
		p.handleAttachments(ctx, gate, peers, msg.Attachments, msg.Mid, replyTo)
	}
}

// routePostback forwards an Instagram button-click to the messenger.
// Instagram postbacks carry a payload string, not a UUID, so they are routed as plain text messages
// rather than interactive callbacks which require an existing message UUID as in_reply_to.
func (p *instagramProvider) routePostback(ctx context.Context, gate *igmodel.InstagramGate, peers peerPair, pb *Postback) {
	if _, err := p.coreMessengerFor(gate).SendText(ctx, &sharedmodel.SendTextRequest{
		DomainID:   gate.DomainID,
		From:       peers.from,
		To:         peers.to,
		Body:       pb.Payload,
		ExternalID: pb.Mid,
	}); err != nil {
		p.logger.Error("send postback as text failed", "payload", pb.Payload, "err", err)
	}
}
