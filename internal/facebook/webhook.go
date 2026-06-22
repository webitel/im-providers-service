package facebook

import (
	"context"
	"fmt"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
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
//	fetch profile → sync contact → route content
func (p *facebookProvider) processMessage(ctx context.Context, gate *fbmodel.FacebookGate, msg Messaging) error {
	psid := msg.Sender.ID
	if psid == "" {
		return nil
	}
	if msg.Message == nil && msg.Postback == nil {
		return nil
	}

	profile, err := p.api.GetUserProfile(ctx, psid, gate.PageToken)
	if err != nil {
		return fmt.Errorf("fetch profile [psid=%s]: %w", psid, err)
	}

	if _, err := p.syncContact(ctx, gate, psid, profile); err != nil {
		return fmt.Errorf("sync contact [psid=%s]: %w", psid, err)
	}

	if msg.Message != nil {
		profile, err := p.api.GetUserProfile(ctx, psid, gate.PageToken)
		if err != nil {
			return fmt.Errorf("fetch profile [psid=%s]: %w", psid, err)
		}
		if _, err := p.syncContact(ctx, gate, psid, profile); err != nil {
			return fmt.Errorf("sync contact [psid=%s]: %w", psid, err)
		}
	}

	peers := peerPair{
		from: sharedmodel.Peer{Sub: psid, Iss: gate.Peer.Iss},
		to:   sharedmodel.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss},
	}

	if msg.Message != nil {
		p.routeMessage(ctx, gate, peers, msg.Message)
	}
	if msg.Postback != nil {
		p.routePostback(ctx, gate, peers, msg.Postback)
	}
	return nil
}

// routeMessage dispatches inbound text and attachment content to the messenger.
// Errors are logged and non-fatal: a single failed delivery must not block others.
func (p *facebookProvider) routeMessage(ctx context.Context, gate *fbmodel.FacebookGate, peers peerPair, msg *InboundMessage) {
	if msg.Text != "" {
		if _, err := p.messenger.SendText(ctx, &sharedmodel.SendTextRequest{
			DomainID: gate.DomainID,
			From:     peers.from,
			To:       peers.to,
			Body:     msg.Text,
		}); err != nil {
			p.logger.Error("send text failed", "err", err)
		}
	}

	if len(msg.Attachments) > 0 {
		p.handleAttachments(ctx, gate, peers, msg.Attachments)
	}
}

// routePostback forwards a Facebook button-click (persistent menu or template button) to the messenger.
// FB postbacks carry a payload string, not a UUID, so they are routed as plain text messages
// rather than interactive callbacks which require an existing message UUID as in_reply_to.
// https://developers.facebook.com/docs/messenger-platform/reference/webhook-events/messaging-postbacks
func (p *facebookProvider) routePostback(ctx context.Context, gate *fbmodel.FacebookGate, peers peerPair, pb *Postback) {
	if _, err := p.messenger.SendText(ctx, &sharedmodel.SendTextRequest{
		DomainID: gate.DomainID,
		From:     peers.from,
		To:       peers.to,
		Body:     pb.Payload,
	}); err != nil {
		p.logger.Error("send postback as text failed", "payload", pb.Payload, "err", err)
	}
}
