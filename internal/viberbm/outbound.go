package viberbm

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// maxFileNameLen bounds the outbound file name attached to a FILE content.
const maxFileNameLen = 25

func (p *viberBMProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	res, err := p.api.SendText(ctx, g.BaseURL, g.APIKey, g.SenderName, to, req.Text, outboundMessageID(req.ID))

	return toResponse(res, to, err)
}

func (p *viberBMProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	url := firstURL(req.Images)
	if url == "" {
		return nil, vibbmmodel.ErrMediaMissing
	}

	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	res, err := p.api.SendImage(ctx, g.BaseURL, g.APIKey, g.SenderName, to, url, req.Text, outboundMessageID(req.ID))

	return toResponse(res, to, err)
}

func (p *viberBMProvider) SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	url := firstURL(req.Documents)
	if url == "" {
		return nil, vibbmmodel.ErrMediaMissing
	}

	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	res, err := p.api.SendFile(ctx, g.BaseURL, g.APIKey, g.SenderName, to, url, documentName(req), outboundMessageID(req.ID))

	return toResponse(res, to, err)
}

// SendTemplate is the cold-start primitive (not part of provider.Sender):
// an approved, business-initiated template for outreach outside an open 24h
// session. templateID must already be approved.
func (p *viberBMProvider) SendTemplate(ctx context.Context, gateID, toSub, templateID, language string, params map[string]string) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, gateID)
	if err != nil {
		return nil, err
	}

	to, err := p.resolveMSISDN(ctx, g, toSub)
	if err != nil {
		return nil, err
	}

	// Ensure the contact + via exist before a cold-start send so a later reply
	// resolves to the same contact identity. Idempotent; failures are non-fatal.
	if _, syncErr := p.syncContact(ctx, g, to, ""); syncErr != nil {
		p.logger.WarnContext(ctx, "viber_bm template: sync contact failed", "gate_id", gateID, "err", syncErr)
	}

	res, err := p.api.SendTemplate(ctx, g.BaseURL, g.APIKey, g.SenderName, to, templateID, language, params, "")

	return toResponse(res, to, err)
}

// resolveMSISDN returns the recipient MSISDN: a UUID sub is resolved to the
// contact subject via the gateway Search RPC, anything else is normalised in
// place (the UUID-shape check avoids misrouting a formatted number).
func (p *viberBMProvider) resolveMSISDN(ctx context.Context, gate *vibbmmodel.ViberBMGate, sub string) (string, error) {
	if _, err := uuid.Parse(sub); err != nil {
		return normalizeMSISDN(sub), nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	resp, err := p.contactClient.SearchContact(authCtx, &contactv1.SearchContactRequest{
		Ids: []string{sub},
	})
	if err != nil {
		return "", fmt.Errorf("resolve msisdn for %s: %w", sub, err)
	}

	items := resp.GetContacts()
	if len(items) == 0 || items[0].GetSubject() == "" {
		return "", fmt.Errorf("resolve msisdn for %s: contact not found or has no subject", sub)
	}

	return normalizeMSISDN(items[0].GetSubject()), nil
}

// normalizeMSISDN strips a leading '+' and surrounding whitespace; InfoBip
// expects a bare international number.
func normalizeMSISDN(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "+")
}

// outboundMessageID echoes the internal message id to InfoBip as the dedup key,
// mapping the zero UUID to "" so omitempty drops it rather than sending a
// constant key for every message.
func outboundMessageID(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}

	return id.String()
}

// toResponse stamps the recipient and provider message id into the response
// metadata for the outbound handler to persist for receipts.
func toResponse(res *sendResult, to string, err error) (*sharedmodel.MessageResponse, error) {
	if err != nil || res == nil {
		return nil, err
	}

	resp := &sharedmodel.MessageResponse{
		ID: res.MessageID,
		MD: map[string]any{
			"recipient_id": to,
		},
	}

	if res.BulkID != "" {
		resp.MD["bulk_id"] = res.BulkID
	}

	return resp, nil
}

func documentName(req *sharedmodel.Message) string {
	name := ""
	if len(req.Documents) > 0 && req.Documents[0] != nil {
		name = req.Documents[0].FileName
	}

	if name == "" {
		name = "file"
	}

	if runes := []rune(name); len(runes) > maxFileNameLen {
		name = string(runes[:maxFileNameLen])
	}

	return name
}

type urlGetter interface {
	GetURL() string
}

func firstURL[T urlGetter](items []T) string {
	if len(items) == 0 {
		return ""
	}

	return items[0].GetURL()
}
