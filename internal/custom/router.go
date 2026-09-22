package custom

import (
	"context"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

// inbound carries everything one webhook message needs, so the content
// handlers do not each grow a different long parameter list.
type inbound struct {
	gate       *custommodel.CustomGate
	peers      peerPair
	msg        *wireMessage
	externalID string
	variables  map[string]string
}

func (p *customProvider) routeMessage(ctx context.Context, in *inbound) {
	msg := in.msg

	switch {
	case msg.Callback != nil:
		p.forwardCallback(ctx, in)

	case msg.File != nil && msg.File.URL != "":
		p.handleFile(ctx, in)

	case msg.Location != nil:
		p.forwardLocation(ctx, in)

	case msg.Contact != nil:
		p.forwardContact(ctx, in)

	case msg.Text != "":
		p.forwardText(ctx, in)

	default:
		p.logger.WarnContext(ctx, "custom message carries no supported content, skipping",
			"gate_id", in.gate.ID,
			"external_id", in.externalID,
		)
	}
}

func (p *customProvider) forwardText(ctx context.Context, in *inbound) {
	if _, err := p.messenger.SendText(ctx, &sharedmodel.SendTextRequest{
		DomainID:          in.gate.DomainID,
		From:              in.peers.from,
		To:                in.peers.to,
		Body:              in.msg.Text,
		ExternalID:        in.externalID,
		ReplyToExternalID: in.msg.ReplyTo,
		Variables:         in.variables,
	}); err != nil {
		p.logger.ErrorContext(ctx, "send text failed", "gate_id", in.gate.ID, "err", err)
	}
}

func (p *customProvider) forwardLocation(ctx context.Context, in *inbound) {
	loc := in.msg.Location

	var name, address *string
	if loc.Title != "" {
		name = &loc.Title
	}

	if loc.Address != "" {
		address = &loc.Address
	}

	if _, err := p.messenger.SendLocation(ctx, &sharedmodel.SendLocationRequest{
		DomainID:          int(in.gate.DomainID),
		From:              in.peers.from,
		To:                in.peers.to,
		Latitude:          loc.Lat,
		Longitude:         loc.Lon,
		Name:              name,
		Address:           address,
		ExternalID:        in.externalID,
		ReplyToExternalID: in.msg.ReplyTo,
	}); err != nil {
		p.logger.ErrorContext(ctx, "send location failed", "gate_id", in.gate.ID, "err", err)
	}
}

func (p *customProvider) forwardContact(ctx context.Context, in *inbound) {
	c := in.msg.Contact

	name := c.Name
	if name == "" {
		name = in.peers.from.Name
	}

	var email *string
	if c.Email != "" {
		email = &c.Email
	}

	if _, err := p.messenger.SendContact(ctx, &sharedmodel.SendContactRequest{
		DomainID:          int(in.gate.DomainID),
		From:              in.peers.from,
		To:                in.peers.to,
		Name:              &name,
		PhoneNumber:       &c.Phone,
		Email:             email,
		ExternalID:        in.externalID,
		ReplyToExternalID: in.msg.ReplyTo,
	}); err != nil {
		p.logger.ErrorContext(ctx, "send contact failed", "gate_id", in.gate.ID, "err", err)
	}
}

// forwardCallback reports a menu button press
func (p *customProvider) forwardCallback(ctx context.Context, in *inbound) {
	cb := in.msg.Callback

	err := p.messenger.SendInteractiveCallback(ctx, &sharedmodel.SendInteractiveCallbackRequest{
		From:         in.peers.from,
		To:           in.peers.to,
		DomainID:     in.gate.DomainID,
		InReplyTo:    cb.MessageID,
		ButtonCode:   cb.Code,
		CallbackData: cb.Data,
	})
	if err == nil {
		return
	}

	p.logger.ErrorContext(ctx, "send interactive callback failed, forwarding as text",
		"gate_id", in.gate.ID,
		"button", cb.Code,
		"err", err,
	)

	fallback := in.msg.Text
	if fallback == "" {
		fallback = cb.Data
	}

	if fallback == "" {
		fallback = cb.Code
	}

	if fallback == "" {
		return
	}

	text := *in.msg
	text.Text = fallback
	fallbackIn := *in
	fallbackIn.msg = &text

	p.forwardText(ctx, &fallbackIn)
}
