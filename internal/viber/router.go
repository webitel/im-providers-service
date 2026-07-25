package viber

import (
	"context"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

// mediaKind distinguishes how downloaded media is forwarded to the core.
type mediaKind int

const (
	mediaImage mediaKind = iota
	mediaDocument
)

// routeMessage dispatches an inbound message by its Viber content type.
// Errors are logged and non-fatal: one failed delivery must not block the rest.
// Keyboard button taps arrive as type "text" with the ActionBody in Text, so they
// are routed as plain text (no message UUID is available for an interactive callback).
func (p *viberProvider) routeMessage(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, msg *inboundMessage, externalID string) {
	switch msg.Type {
	case "text", "url":
		text := msg.Text
		if text == "" && msg.Type == "url" {
			text = msg.Media
		}
		if text == "" {
			return
		}
		if _, err := p.messenger.SendText(ctx, &sharedmodel.SendTextRequest{
			DomainID:   gate.DomainID,
			From:       peers.from,
			To:         peers.to,
			Body:       text,
			ExternalID: externalID,
		}); err != nil {
			p.logger.Error("send text failed", "err", err)
		}

	case "picture":
		p.handleMedia(ctx, gate, peers, msg, externalID, mediaImage)

	case "video", "file":
		p.handleMedia(ctx, gate, peers, msg, externalID, mediaDocument)

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
		p.logger.Debug("viber sticker message not supported, skipping", "external_id", externalID)

	default:
		p.logger.Warn("unsupported viber message type, skipping", "type", msg.Type)
	}
}
