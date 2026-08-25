package instagram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

func (p *instagramProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	igsid, err := p.resolveIGSID(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	if len(req.Text) > 1000 {
		return nil, errors.New("instagram: message text exceeds 1000 characters")
	}

	if !p.withinMessageWindow(ctx, req.GateID, igsid) {
		return p.sendHumanAgentText(ctx, g.IGToken, igsid, req.Text, igsid)
	}

	return withRecipient(p.api.SendText(ctx, g.IGToken, igsid, req.Text))(igsid)
}

func (p *instagramProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	igsid, err := p.resolveIGSID(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	return withRecipient(p.api.SendMedia(ctx, g.IGToken, igsid, MediaImage, firstURL(req.Images)))(igsid)
}

func (p *instagramProvider) SendDocument(_ context.Context, _ *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return nil, errors.New("instagram: document upload not supported by Instagram Direct Messaging API")
}

func (p *instagramProvider) SendInteractive(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}

	igsid, err := p.resolveIGSID(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}

	return withRecipient(p.api.SendInteractive(ctx, g.IGToken, igsid, req.Text, req.Interactive))(igsid)
}

// resolveIGSID returns the Instagram IGSID for the given sub.
// If sub is already a numeric IGSID it is returned as-is; otherwise it is
// treated as an internal contact UUID and resolved via the gateway Search RPC.
func (p *instagramProvider) resolveIGSID(ctx context.Context, gate *igmodel.InstagramGate, contactID string) (string, error) {
	if !strings.Contains(contactID, "-") {
		return contactID, nil
	}

	if igsid, ok, _ := p.igsidCache.Get(ctx, contactID); ok {
		return igsid, nil
	}

	authCtx := withGatewayIdentity(ctx, gate)

	resp, err := p.contactClient.SearchContact(authCtx, &contactv1.SearchContactRequest{
		Ids: []string{contactID},
	})
	if err != nil {
		return "", fmt.Errorf("resolve igsid for %s: %w", contactID, err)
	}

	items := resp.GetContacts()

	if len(items) == 0 || items[0].GetSubject() == "" {
		return "", fmt.Errorf("resolve igsid for %s: contact not found or has no subject", contactID)
	}

	igsid := items[0].GetSubject()
	_ = p.igsidCache.Set(ctx, contactID, igsid)

	return igsid, nil
}

// withRecipient stamps the resolved IGSID into the response metadata so the
// outbound handler can persist it for receipt correlation.
func withRecipient(resp *sharedmodel.MessageResponse, err error) func(igsid string) (*sharedmodel.MessageResponse, error) {
	return func(igsid string) (*sharedmodel.MessageResponse, error) {
		if err != nil || resp == nil {
			return resp, err
		}

		if resp.MD == nil {
			resp.MD = make(map[string]any, 1)
		}

		resp.MD["recipient_id"] = igsid

		return resp, nil
	}
}

type urlGetter interface {
	GetURL() string
}

// firstURL returns the URL of the first element, or "" if the slice is empty.
func firstURL[T urlGetter](items []T) string {
	if len(items) == 0 {
		return ""
	}

	return items[0].GetURL()
}

// sendHumanAgentText sends a text message using MESSAGE_TAG with HUMAN_AGENT tag
// when outside the 24-hour messaging window.
func (p *instagramProvider) sendHumanAgentText(ctx context.Context, token, igsid, text, recipient string) (*sharedmodel.MessageResponse, error) {
	resp, err := p.api.SendHumanAgentText(ctx, token, igsid, text)

	return withRecipient(resp, err)(recipient)
}
