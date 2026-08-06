package facebook

import (
	"context"
	"fmt"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
)

func (p *facebookProvider) HandleWebhook(ctx context.Context, data []byte) error {
	evt, err := p.api.ParseWebhook(data)
	if err != nil || evt == nil || len(evt.Entry) == 0 {
		return nil
	}

	gate, err := p.resolveGate(ctx, p.webhookURI(ctx), evt.Entry[0].ID)
	if err != nil || !gate.Enabled {
		return err
	}

	for _, msg := range evt.AllMessages() {
		if err := p.processMessage(ctx, gate, msg); err != nil {
			p.logger.Error("message dropped", "sender", msg.Sender.ID, "err", err)
		}
	}
	return nil
}

// processMessage is the per-event pipeline:
//
//	dedup → fetch profile → sync contact → route content
func (p *facebookProvider) processMessage(ctx context.Context, gate *fbmodel.FacebookGate, msg Messaging) error {
	psid := msg.Sender.ID
	if psid == "" {
		return nil
	}
	if msg.Delivery != nil || msg.Read != nil {
		p.processReceipt(ctx, gate, psid, msg)

		return nil
	}
	if msg.Message == nil && msg.Postback == nil {
		return nil
	}
	if msg.Message != nil && msg.Message.IsEcho {
		return nil
	}

	// Deduplicate: Facebook may deliver the same event more than once.
	mid := messageID(msg)
	if mid != "" && messageSeen(ctx, p.rdb, mid) {
		p.logger.DebugContext(ctx, "duplicate facebook event skipped", "mid", mid)
		return nil
	}

	profile, err := p.api.GetUserProfile(ctx, psid, gate.PageToken)
	if err != nil {
		return fmt.Errorf("fetch profile [psid=%s]: %w", psid, err)
	}

	if _, err := p.syncContact(ctx, gate, psid, profile); err != nil {
		return fmt.Errorf("sync contact [psid=%s]: %w", psid, err)
	}

	peers := peerPair{
		from: sharedmodel.Peer{Sub: psid, Iss: gate.Peer.Iss},
		to:   sharedmodel.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss, Via: &gate.ID},
	}

	p.logger.DebugContext(ctx, "facebook inbound peers built",
		"from_sub", peers.from.Sub,
		"from_iss", peers.from.Iss,
		"to_sub", peers.to.Sub,
		"to_iss", peers.to.Iss,
		"to_via", gate.ID,
	)

	if msg.Message != nil {
		p.routeMessage(ctx, gate, peers, msg.Message)
	}
	if msg.Postback != nil {
		p.routePostback(ctx, gate, peers, msg.Postback)
	}
	return nil
}

// processReceipt maps Facebook delivery/read receipts to delivery-status
// reports for im-thread-service. Delivery receipts with explicit mids are
// resolved directly; watermark-only receipts confirm every message sent to
// the user before the watermark.
func (p *facebookProvider) processReceipt(ctx context.Context, gate *fbmodel.FacebookGate, psid string, msg Messaging) {
	if d := msg.Delivery; d != nil {
		at := watermarkTime(d.Watermark, msg.Timestamp)

		if len(d.Mids) > 0 {
			p.status.DeliveredByProviderIDs(ctx, gate.ID, d.Mids, at)
		} else {
			p.status.DeliveredUpTo(ctx, gate.ID, psid, at)
		}
	}

	if r := msg.Read; r != nil {
		p.status.ReadUpTo(ctx, gate.ID, psid, watermarkTime(r.Watermark, msg.Timestamp))
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

// messageID returns the Facebook message or postback mid for deduplication.
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
func (p *facebookProvider) routeMessage(ctx context.Context, gate *fbmodel.FacebookGate, peers peerPair, msg *InboundMessage) {
	replyTo := ""
	if msg.ReplyTo != nil {
		replyTo = msg.ReplyTo.Mid
	}

	if msg.Text != "" {
		if _, err := p.messenger.SendText(ctx, &sharedmodel.SendTextRequest{
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

// routePostback forwards a Facebook button-click (persistent menu or template button) to the messenger.
// FB postbacks carry a payload string, not a UUID, so they are routed as plain text messages
// rather than interactive callbacks which require an existing message UUID as in_reply_to.
// https://developers.facebook.com/docs/messenger-platform/reference/webhook-events/messaging-postbacks
func (p *facebookProvider) routePostback(ctx context.Context, gate *fbmodel.FacebookGate, peers peerPair, pb *Postback) {
	if _, err := p.messenger.SendText(ctx, &sharedmodel.SendTextRequest{
		DomainID:   gate.DomainID,
		From:       peers.from,
		To:         peers.to,
		Body:       pb.Payload,
		ExternalID: pb.Mid,
	}); err != nil {
		p.logger.Error("send postback as text failed", "payload", pb.Payload, "err", err)
	}
}
