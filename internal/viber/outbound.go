package viber

import (
	"context"
	"fmt"
	"strings"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

// Viber implements the outbound location/contact senders in addition to the base Sender.
var (
	_ provider.LocationSender = (*viberProvider)(nil)
	_ provider.ContactSender  = (*viberProvider)(nil)
)

func (p *viberProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return p.api.SendText(ctx, g.AuthToken, senderOf(g), receiver, req.Text, nil)
}

func (p *viberProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	if len(req.Images) == 0 || req.Images[0] == nil || req.Images[0].URL == "" {
		return nil, fmt.Errorf("viber: image url missing")
	}
	return p.api.SendPicture(ctx, g.AuthToken, senderOf(g), receiver, req.Images[0].URL, req.Text)
}

func (p *viberProvider) SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	if len(req.Documents) == 0 || req.Documents[0] == nil || req.Documents[0].URL == "" {
		return nil, fmt.Errorf("viber: document url missing")
	}
	d := req.Documents[0]
	size := d.Size
	if size <= 0 {
		size = 1
	}
	return p.api.SendFile(ctx, g.AuthToken, senderOf(g), receiver, d.URL, d.FileName, size)
}

func (p *viberProvider) SendInteractive(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	kb := buildKeyboard(req.Interactive)
	body := req.Text
	if body == "" && req.Interactive != nil {
		body = req.Interactive.Body
	}
	return p.api.SendText(ctx, g.AuthToken, senderOf(g), receiver, body, kb)
}

func (p *viberProvider) SendLocation(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if req.Location == nil {
		return nil, fmt.Errorf("viber: location payload missing")
	}
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return p.api.SendLocation(ctx, g.AuthToken, senderOf(g), receiver, req.Location.Latitude, req.Location.Longitude)
}

func (p *viberProvider) SendContact(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if req.Contact == nil {
		return nil, fmt.Errorf("viber: contact payload missing")
	}
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return p.api.SendContact(ctx, g.AuthToken, senderOf(g), receiver, req.Contact.Name, req.Contact.PhoneNumber)
}

// prepare loads the gate and resolves the recipient's Viber user id.
func (p *viberProvider) prepare(ctx context.Context, gateID, toSub string) (*vibmodel.ViberGate, string, error) {
	g, err := p.fetchGate(ctx, gateID)
	if err != nil {
		return nil, "", err
	}
	receiver, err := p.resolveReceiver(ctx, g, toSub)
	if err != nil {
		return nil, "", err
	}
	return g, receiver, nil
}

// resolveReceiver returns the Viber user id for the given sub. A sub that is already a
// native Viber id (opaque base64, never contains "-") is returned as-is; otherwise it is
// treated as an internal contact UUID and resolved via the contact Search RPC.
func (p *viberProvider) resolveReceiver(ctx context.Context, gate *vibmodel.ViberGate, contactID string) (string, error) {
	if !strings.Contains(contactID, "-") {
		return contactID, nil
	}
	if id, ok, _ := p.receiverCache.Get(ctx, contactID); ok {
		return id, nil
	}
	authCtx := withGatewayIdentity(ctx, gate)
	resp, err := p.contactClient.SearchContact(authCtx, &contactv1.SearchContactRequest{
		Ids: []string{contactID},
	})
	if err != nil {
		return "", fmt.Errorf("resolve viber id for %s: %w", contactID, err)
	}
	items := resp.GetContacts()
	if len(items) == 0 || items[0].GetSubject() == "" {
		return "", fmt.Errorf("resolve viber id for %s: contact not found or has no subject", contactID)
	}
	id := items[0].GetSubject()
	_ = p.receiverCache.Set(ctx, contactID, id)
	return id, nil
}
