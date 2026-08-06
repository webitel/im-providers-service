package facebook

import (
	"context"
	"fmt"
	"strings"

	contactv1 "github.com/webitel/im-providers-service/gen/go/contact/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	"github.com/webitel/im-providers-service/internal/provider"
)

func (p *facebookProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	psid, err := p.resolvePSID(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return withRecipient(p.api.SendText(ctx, g.PageToken, psid, req.Text))(psid)
}

func (p *facebookProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	psid, err := p.resolvePSID(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return withRecipient(p.api.SendMedia(ctx, g.PageToken, psid, MediaImage, firstURL(req.Images)))(psid)
}

func (p *facebookProvider) SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	psid, err := p.resolvePSID(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return withRecipient(p.api.SendMedia(ctx, g.PageToken, psid, MediaFile, firstURL(req.Documents)))(psid)
}

func (p *facebookProvider) SendInteractive(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	psid, err := p.resolvePSID(ctx, g, req.To.Sub)
	if err != nil {
		return nil, err
	}
	return withRecipient(p.api.SendInteractive(ctx, g.PageToken, psid, req.Text, req.Interactive))(psid)
}

// SendTyping forwards a Messenger sender_action (typing_on/typing_off) to the
// external chat partner. Best-effort: it returns an error to the caller but
// persists nothing.
func (p *facebookProvider) SendTyping(ctx context.Context, req *provider.TypingRequest) error {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return err
	}

	psid, err := p.resolvePSID(ctx, g, req.ExternalID)
	if err != nil {
		return err
	}

	return p.api.SendTyping(ctx, g.PageToken, psid, req.TypingOn)
}

// resolvePSID returns the Facebook PSID for the given sub.
// If sub is already a numeric PSID it is returned as-is; otherwise it is
// treated as an internal contact UUID and resolved via the gateway Search RPC.
func (p *facebookProvider) resolvePSID(ctx context.Context, gate *fbmodel.FacebookGate, contactID string) (string, error) {
	if !strings.Contains(contactID, "-") {
		return contactID, nil
	}
	if psid, ok, _ := p.psidCache.Get(ctx, contactID); ok {
		return psid, nil
	}
	authCtx := withGatewayIdentity(ctx, gate)
	resp, err := p.contactClient.SearchContact(authCtx, &contactv1.SearchContactRequest{
		Ids: []string{contactID},
	})
	if err != nil {
		return "", fmt.Errorf("resolve psid for %s: %w", contactID, err)
	}
	items := resp.GetContacts()
	if len(items) == 0 || items[0].GetSubject() == "" {
		return "", fmt.Errorf("resolve psid for %s: contact not found or has no subject", contactID)
	}
	psid := items[0].GetSubject()
	_ = p.psidCache.Set(ctx, contactID, psid)
	return psid, nil
}

// withRecipient stamps the resolved PSID into the response metadata so the
// outbound handler can persist it for watermark-based webhook receipts.
func withRecipient(resp *sharedmodel.MessageResponse, err error) func(psid string) (*sharedmodel.MessageResponse, error) {
	return func(psid string) (*sharedmodel.MessageResponse, error) {
		if err != nil || resp == nil {
			return resp, err
		}

		if resp.MD == nil {
			resp.MD = make(map[string]any, 1)
		}

		resp.MD["recipient_id"] = psid

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
