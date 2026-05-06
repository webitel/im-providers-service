package facebook

import (
	"context"

	"github.com/webitel/im-providers-service/internal/domain/model"
)

func (p *facebookProvider) HandleWebhook(ctx context.Context, data []byte) error {
	uri := p.normalizeURI(ctx)
	req, err := p.api.ParseWebhook(data)
	if err != nil || req == nil || len(req.Entry) == 0 {
		return nil
	}

	gate, err := p.resolveGate(ctx, uri, req.Entry[0].ID)
	if err != nil || !gate.Enabled {
		return err
	}

	for _, m := range req.AllMessages() {
		psid := m.Sender.ID
		if psid == "" {
			continue
		}

		fbusr, err := p.api.GetUserProfile(ctx, psid, gate.PageToken)
		if err != nil {
			p.logger.Error("failed to fetch user profile", "psid", psid, "err", err)
			continue
		}

		contact, err := p.externalUserSync(ctx, gate, psid, fbusr)
		if err != nil {
			p.logger.Error("sync failed", "psid", psid, "err", err)
			continue
		}

		from := model.Peer{Sub: contact.Sub, Iss: gate.Peer.Iss}
		to := model.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss}

		if m.Message != nil {
			if m.Message.Text != "" {
				if _, err := p.messenger.SendText(ctx, &model.SendTextRequest{
					From:     from,
					To:       to,
					Body:     m.Message.Text,
					DomainID: gate.DomainID,
				}); err != nil {
					p.logger.Error("failed to send text", "psid", psid, "err", err)
				}
			}

			if len(m.Message.Attachments) > 0 {
				p.handleAttachments(ctx, gate, from, to, m.Message.Attachments)
			}
		}
	}
	return nil
}
