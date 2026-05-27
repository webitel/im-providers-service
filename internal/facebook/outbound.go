package facebook

import (
	"context"
	"fmt"
	"strings"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
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
	return p.api.SendText(ctx, g.PageToken, psid, req.Text)
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
	return p.api.SendMedia(ctx, g.PageToken, psid, MediaImage, firstURL(req.Images))
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
	return p.api.SendMedia(ctx, g.PageToken, psid, MediaFile, firstURL(req.Documents))
}

// resolvePSID returns the Facebook PSID for the given sub.
// If sub is already a numeric PSID it is returned as-is; otherwise it is
// treated as an internal contact UUID and resolved via the gateway Locate RPC.
func (p *facebookProvider) resolvePSID(ctx context.Context, gate *fbmodel.FacebookGate, sub string) (string, error) {
	if !strings.Contains(sub, "-") {
		return sub, nil
	}
	if psid, ok := p.psidCache.Get(sub); ok {
		return psid, nil
	}
	authCtx := withGatewayIdentity(ctx, gate)
	resp, err := p.gatewayer.Locate(authCtx, &gatewayv1.LocateConatctRequest{
		Id:       sub,
		DomainId: gate.DomainID,
	})
	if err != nil {
		return "", fmt.Errorf("resolve psid for %s: %w", sub, err)
	}
	psid := resp.GetItem().GetSub()
	if psid == "" {
		return "", fmt.Errorf("resolve psid for %s: contact has no sub", sub)
	}
	p.psidCache.Add(sub, psid)
	return psid, nil
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
