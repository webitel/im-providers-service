package viber

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

var (
	_ provider.LocationSender = (*viberProvider)(nil)
	_ provider.ContactSender  = (*viberProvider)(nil)
)

func (p *viberProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return p.api.SendText(ctx, g.AuthToken, senderOf(g, req.SenderName), receiver, req.Text, nil)
}

func (p *viberProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if len(req.Images) == 0 || req.Images[0] == nil || req.Images[0].URL == "" {
		return nil, fmt.Errorf("viber: image url missing")
	}
	img := req.Images[0]
	return p.dispatchMedia(ctx, req, attachment{
		url:  img.URL,
		name: img.FileName,
		mime: img.MimeType,
		size: img.Size,
	})
}

func (p *viberProvider) SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if len(req.Documents) == 0 || req.Documents[0] == nil || req.Documents[0].URL == "" {
		return nil, fmt.Errorf("viber: document url missing")
	}
	d := req.Documents[0]
	return p.dispatchMedia(ctx, req, attachment{
		url:  d.URL,
		name: d.FileName,
		mime: d.MimeType,
		size: d.Size,
	})
}

type attachment struct {
	url  string
	name string
	mime string
	size int64
}

func (p *viberProvider) dispatchMedia(ctx context.Context, req *sharedmodel.Message, att attachment) (*sharedmodel.MessageResponse, error) {
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}

	p.probeMedia(ctx, att.url, &att.mime, &att.size)

	if att.size > maxFileBytes {
		return nil, fmt.Errorf("viber: attachment %d bytes exceeds the platform limit of %d", att.size, int64(maxFileBytes))
	}

	kind := classify(att.mime, att.name)
	if kind == sendVideo && att.size > maxVideoBytes {
		kind = sendFile
	}
	if kind == sendGIF && len(att.url) > maxMediaURLLen {
		kind = sendFile
	}

	if req.Text != "" && kind != sendPicture {
		if _, err := p.api.SendText(ctx, g.AuthToken, senderOf(g, req.SenderName), receiver, req.Text, nil); err != nil {
			p.logger.WarnContext(ctx, "viber caption send failed", "gate_id", g.ID, "err", err)
		}
	}

	resp, err := p.sendAs(ctx, g, receiver, req.SenderName, kind, req.Text, att)
	if err == nil || kind == sendFile {
		return resp, err
	}

	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return nil, err
	}

	p.logger.WarnContext(ctx, "viber media type refused, retrying as file",
		"gate_id", g.ID,
		"kind", kind.String(),
		"status", apiErr.Status,
		"err", err,
	)

	return p.sendAs(ctx, g, receiver, req.SenderName, sendFile, req.Text, att)
}

func (p *viberProvider) sendAs(ctx context.Context, g *vibmodel.ViberGate, receiver, senderName string, kind sendKind, caption string, att attachment) (*sharedmodel.MessageResponse, error) {
	switch kind {
	case sendPicture:
		return p.api.SendPicture(ctx, g.AuthToken, senderOf(g, senderName), receiver, att.url, caption)
	case sendVideo:
		return p.api.SendVideo(ctx, g.AuthToken, senderOf(g, senderName), receiver, att.url, fileSize(att.size), 0)
	case sendGIF:
		return p.api.SendURL(ctx, g.AuthToken, senderOf(g, senderName), receiver, att.url)
	case sendFile:
		return p.api.SendFile(ctx, g.AuthToken, senderOf(g, senderName), receiver, att.url, fileName(att), fileSize(att.size))
	default:
		return p.api.SendFile(ctx, g.AuthToken, senderOf(g, senderName), receiver, att.url, fileName(att), fileSize(att.size))
	}
}

func fileSize(size int64) int64 {
	if size <= 0 {
		return 1
	}
	return size
}

func fileName(att attachment) string {
	if att.name != "" {
		return att.name
	}
	if base := path.Base(att.url); base != "" && base != "." && base != "/" {
		return base
	}
	return "file"
}

func (p *viberProvider) SendInteractive(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}

	body := req.Text
	if body == "" && req.Interactive != nil {
		body = req.Interactive.Body
	}

	trackingData := ""
	if req.ID != uuid.Nil {
		trackingData = req.ID.String()
	}

	if req.Interactive != nil && req.Interactive.Placement == sharedmodel.MenuPlacementInline {
		if rm := buildRichMedia(req.Interactive); rm != nil {
			if body != "" {
				if _, err := p.api.SendText(ctx, g.AuthToken, senderOf(g, req.SenderName), receiver, body, nil); err != nil {
					p.logger.WarnContext(ctx, "viber interactive body send failed", "gate_id", g.ID, "err", err)
				}
			}

			resp, err := p.api.SendRichMedia(ctx, g.AuthToken, senderOf(g, req.SenderName), receiver, rm, altTextFor(body, rm), trackingData)
			if err == nil {
				p.rememberMenu(ctx, g.ID, receiver, trackingData, req.Interactive)
			}

			return resp, err
		}
	}

	resp, err := p.api.SendMenu(ctx, g.AuthToken, senderOf(g, req.SenderName), receiver, body, buildKeyboard(req.Interactive), trackingData)
	if err == nil {
		p.rememberMenu(ctx, g.ID, receiver, trackingData, req.Interactive)
	}

	return resp, err
}

func (p *viberProvider) SendLocation(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if req.Location == nil {
		return nil, fmt.Errorf("viber: location payload missing")
	}
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return p.api.SendLocation(ctx, g.AuthToken, senderOf(g, req.SenderName), receiver, req.Location.Latitude, req.Location.Longitude)
}

func (p *viberProvider) SendContact(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	if req.Contact == nil {
		return nil, fmt.Errorf("viber: contact payload missing")
	}
	g, receiver, err := p.prepare(ctx, req.GateID, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return p.api.SendContact(ctx, g.AuthToken, senderOf(g, req.SenderName), receiver, req.Contact.Name, req.Contact.PhoneNumber)
}

func (p *viberProvider) prepare(ctx context.Context, gateID, toSub string) (*vibmodel.ViberGate, string, error) {
	g, err := p.fetchGate(ctx, gateID)
	if err != nil {
		return nil, "", err
	}
	receiver, err := p.resolveReceiver(ctx, g, toSub)
	if err != nil {
		return nil, "", err
	}

	if p.unreachable(ctx, g.ID, receiver) {
		return nil, "", vibmodel.ErrReceiverNotSubscribed
	}

	return g, receiver, nil
}

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
