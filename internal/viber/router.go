package viber

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

func (p *viberProvider) routeMessage(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, msg *inboundMessage, externalID string) {
	switch msg.Type {
	case "text":
		if btn, messageID, ok := p.resolveMenuTap(ctx, gate.ID, peers.from.Sub, msg); ok {
			if err := p.forwardMenuTap(ctx, gate, peers, btn, messageID); err != nil {
				p.logger.Error("send interactive callback failed, forwarding as text",
					"gate_id", gate.ID,
					"button", btn.Code,
					"err", err,
				)
				p.forwardText(ctx, gate, peers, msg.Text, externalID)
			}

			return
		}

		p.forwardText(ctx, gate, peers, msg.Text, externalID)

	case "url":
		p.handleURL(ctx, gate, peers, msg, externalID)

	case "picture", "video", "file":
		p.handleMedia(ctx, gate, peers, msg, externalID)

	case "location":
		if msg.Location == nil {
			return
		}
		if _, err := p.messenger.SendLocation(ctx, &sharedmodel.SendLocationRequest{
			DomainID:   int(gate.DomainID),
			From:       peers.from,
			To:         peers.to,
			Latitude:   msg.Location.Lat,
			Longitude:  msg.Location.Lon,
			ExternalID: externalID,
		}); err != nil {
			p.logger.Error("send location failed", "err", err)
		}

	case "contact":
		if msg.Contact == nil {
			return
		}
		name := msg.Contact.Name
		if name == "" {
			name = peers.from.Name
		}
		phone := msg.Contact.PhoneNumber
		if _, err := p.messenger.SendContact(ctx, &sharedmodel.SendContactRequest{
			DomainID:    int(gate.DomainID),
			From:        peers.from,
			To:          peers.to,
			Name:        &name,
			PhoneNumber: &phone,
			ExternalID:  externalID,
		}); err != nil {
			p.logger.Error("send contact failed", "err", err)
		}

	case "sticker":
		p.forwardText(ctx, gate, peers, stickerPlaceholder(msg.StickerID), externalID)

	default:
		p.logger.Warn("unsupported viber message type, skipping", "type", msg.Type)
	}
}

func (p *viberProvider) forwardText(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, body, externalID string) {
	if body == "" {
		return
	}
	if _, err := p.messenger.SendText(ctx, &sharedmodel.SendTextRequest{
		DomainID:   gate.DomainID,
		From:       peers.from,
		To:         peers.to,
		Body:       body,
		ExternalID: externalID,
	}); err != nil {
		p.logger.Error("send text failed", "err", err)
	}
}

func (p *viberProvider) handleURL(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, msg *inboundMessage, externalID string) {
	link := msg.Media
	if link == "" {
		link = msg.Text
	}
	if link == "" {
		return
	}

	if p.isFetchableMedia(ctx, link) {
		fetch := *msg
		fetch.Media = link
		fetch.FileName = fileNameFromURL(link)

		media, err := p.syncMedia(ctx, p.linkClient, gate.DomainID, &fetch)
		if err == nil {
			p.forwardMedia(ctx, gate, peers, media, externalID)
			return
		}

		p.logger.WarnContext(ctx, "viber url media fetch failed, forwarding as text", "url", link, "err", err)
	}

	body := msg.Text
	if body == "" {
		body = link
	}
	p.forwardText(ctx, gate, peers, body, externalID)
}

func (p *viberProvider) isFetchableMedia(ctx context.Context, link string) bool {
	if !strings.HasPrefix(strings.ToLower(link), "https://") {
		return false
	}

	contentType, size, ok := p.probeLink(ctx, link)
	if !ok || size > maxInboundBytes {
		return false
	}

	return contentType == "image/gif" || strings.HasPrefix(contentType, "video/")
}

func fileNameFromURL(link string) string {
	parsed, err := url.Parse(link)
	if err != nil {
		return ""
	}

	base := path.Base(parsed.Path)
	if base == "." || base == "/" || filepath.Ext(base) == "" {
		return ""
	}

	return base
}

func stickerPlaceholder(id int64) string {
	if id == 0 {
		return "🙂 sticker"
	}
	return fmt.Sprintf("🙂 sticker #%d", id)
}
