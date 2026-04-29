package facebook

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/provider/facebook/graph"
	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/im-providers-service/internal/store"
)

type facebookProvider struct {
	api       *Client
	logger    *slog.Logger
	messenger service.Messenger
	gateCache store.GateCache
	userCache store.ExternalUserCache
	repo      store.FacebookStore
	gatewayer gatewayv1.ContactsClient
}

func New(m service.Messenger, l *slog.Logger, gc store.GateCache, uc store.ExternalUserCache, repo store.FacebookStore, core gatewayv1.ContactsClient) provider.Provider {
	return &facebookProvider{
		api:       NewClient(l),
		logger:    l.With("provider", "facebook"),
		messenger: m,
		gateCache: gc,
		userCache: uc,
		repo:      repo,
		gatewayer: core,
	}
}

func (p *facebookProvider) Type() string { return "facebook" }

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
			p.logger.Error("profile fetch failed", "psid", psid, "err", err)
			continue
		}

		p.logDebug(uri, req.Entry[0].ID, psid, fbusr)

		if err := p.usrsync(ctx, gate, psid, fbusr); err != nil {
			p.logger.Error("sync failed", "psid", psid, "err", err)
			continue
		}

		if m.Message != nil && m.Message.Text != "" {
			p.messenger.SendText(ctx, &model.SendTextRequest{
				From: model.Peer{Sub: psid, Iss: gate.Peer.Iss},
				To:   model.Peer{Sub: gate.Peer.Sub, Iss: gate.Peer.Iss},
				Body: m.Message.Text,
			})
		}
	}
	return nil
}

func (p *facebookProvider) SendText(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.repo.Select(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	return p.api.SendText(ctx, g.PageToken, req.To.Sub, req.Text)
}

func (p *facebookProvider) SendImage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.repo.Select(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	var u string
	if len(req.Images) > 0 {
		u = req.Images[0].URL
	}
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaImage, u)
}

func (p *facebookProvider) SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.repo.Select(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	var u string
	if len(req.Documents) > 0 {
		u = req.Documents[0].URL
	}
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaFile, u)
}

// Verify implements the provider.Verifier interface for Facebook's subscription handshake.
func (p *facebookProvider) Verify(ctx context.Context, query url.Values) (string, error) {
	// Call the parser from the metadata package
	req := graph.ParseVerify(query)

	if req.Mode != "subscribe" {
		return "", fmt.Errorf("unexpected hub.mode: %s", req.Mode)
	}

	// Logic: Fetch the gate from DB using the URI to validate the specific token
	// This ensures only Meta servers knowing your secret can verify the webhook.
	if req.VerifyToken == "" {
		return "", fmt.Errorf("missing verify_token")
	}

	p.logger.Info("webhook verified successfully", "mode", req.Mode)
	return req.Challenge, nil
}

func (p *facebookProvider) usrsync(ctx context.Context, g *model.FacebookGate, psid string, prof *UserProfile) error {
	u := &model.ExternalUser{ID: psid, FirstName: prof.FirstName, LastName: prof.LastName}
	if ok, _ := p.userCache.IsKnown(ctx, u); ok {
		return nil
	}
	_, err := p.gatewayer.Create(ctx, &gatewayv1.CreateContactRequest{
		IssId:    g.Peer.Iss,
		Type:     p.Type(),
		Name:     u.FirstName,
		Username: u.LastName,
		Subject:  u.ID,
		IsBot:    false,
	})
	if err == nil {
		_ = p.userCache.MarkKnown(ctx, u)
	}
	return err
}

func (p *facebookProvider) resolveGate(ctx context.Context, uri, pageID string) (*model.FacebookGate, error) {
	k := uri + ":" + pageID
	if _, ok := p.gateCache.Get(k); ok {
		return p.repo.SelectByPageAndURI(ctx, pageID, uri)
	}
	g, err := p.repo.SelectByPageAndURI(ctx, pageID, uri)
	if err != nil {
		return nil, err
	}
	p.gateCache.Set(k, store.GateState{GateID: g.ID, Enabled: g.Enabled, Issuer: g.Peer.Iss, Sub: g.Peer.Sub})
	return g, nil
}

func (p *facebookProvider) logDebug(uri, pageID, psid string, prof *UserProfile) {
	fmt.Printf("\033[35m[FB]\033[0m %s (Page: %s) | \033[32m%s %s\033[0m (ID: %s)\n", uri, pageID, prof.FirstName, prof.LastName, psid)
}

func (p *facebookProvider) normalizeURI(ctx context.Context) string {
	uri, _ := ctx.Value("webhook_uri").(string)
	if !strings.HasPrefix(uri, "/") {
		return "/" + uri
	}
	return uri
}
