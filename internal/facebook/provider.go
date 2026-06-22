// Package facebook implements the Messenger Platform provider.
// https://developers.facebook.com/docs/messenger-platform
package facebook

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	"github.com/webitel/im-providers-service/internal/provider"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
)

type facebookProvider struct {
	api         graphAPI
	logger      *slog.Logger
	messenger   sharedsvc.Messenger
	gateCache   sharedstore.GateCache
	userCache   sharedstore.ExternalUserCache
	repo        fbstore.FacebookStore
	metaAppRepo fbstore.MetaAppStore
	gatewayer   *imgateway.Client
	media       sharedsvc.MediaManager
	contactClient *imcontact.Client
	// psidCache maps internal contact UUID → Facebook PSID to avoid an
	// im-contact round-trip on every outbound message.
	psidCache *lru.Cache[string, string]
	// httpClient is used exclusively for media downloads; kept separate from
	// api.http so the two timeouts can be tuned independently.
	httpClient *http.Client
}

func New(
	m sharedsvc.Messenger,
	l *slog.Logger,
	gc sharedstore.GateCache,
	uc sharedstore.ExternalUserCache,
	repo fbstore.FacebookStore,
	metaAppRepo fbstore.MetaAppStore,
	gatewayer *imgateway.Client,
	media sharedsvc.MediaManager,
	contactClient *imcontact.Client,
	api *apiClient,
) provider.Provider {
	psidCache, _ := lru.New[string, string](1000)
	return &facebookProvider{
		api:           api,
		logger:        l.With("provider", "facebook"),
		messenger:     m,
		gateCache:     gc,
		userCache:     uc,
		repo:          repo,
		metaAppRepo:   metaAppRepo,
		gatewayer:     gatewayer,
		media:         media,
		contactClient: contactClient,
		psidCache:     psidCache,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

var _ provider.InteractiveSender = (*facebookProvider)(nil)

func (p *facebookProvider) Type() string { return "facebook" }

func (p *facebookProvider) Verify(ctx context.Context, query url.Values) (string, error) {
	req := parseVerify(query)
	if req.Mode != "subscribe" {
		return "", fmt.Errorf("unexpected hub.mode: %s", req.Mode)
	}

	uri := p.webhookURI(ctx)
	app, err := p.metaAppRepo.SelectByURI(ctx, uri)
	if err != nil {
		return "", fmt.Errorf("verify: app lookup failed: %w", err)
	}

	if app.VerifyToken != "" && req.VerifyToken != app.VerifyToken {
		return "", fmt.Errorf("verify_token mismatch")
	}

	return req.Challenge, nil
}

// resolveGate returns the FacebookGate for the given page. Disabled gates are
// short-circuited from the LRU cache to avoid an unnecessary DB round-trip on
// every webhook delivery.
func (p *facebookProvider) resolveGate(ctx context.Context, uri, pageID string) (*fbmodel.FacebookGate, error) {
	k := gateKey(uri, pageID)
	if cached, ok := p.gateCache.Get(k); ok && !cached.Enabled {
		return &fbmodel.FacebookGate{Enabled: false}, nil
	}

	g, err := p.repo.SelectByPageAndURI(ctx, pageID, uri)
	if err != nil {
		return nil, err
	}
	p.gateCache.Set(k, sharedstore.GateState{
		GateID:  g.ID,
		Enabled: g.Enabled,
		Issuer:  g.Peer.Iss,
		Sub:     g.Peer.Sub,
		Domain:  g.DomainID,
	})
	return g, nil
}

func (p *facebookProvider) fetchGate(ctx context.Context, gateID string) (*fbmodel.FacebookGate, error) {
	return p.repo.Select(ctx, gateID)
}

// webhookURI extracts and normalises the webhook path segment from context.
func (p *facebookProvider) webhookURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string)
	if !strings.HasPrefix(uri, "/") {
		return "/" + uri
	}
	return uri
}

func gateKey(uri, pageID string) string { return uri + ":" + pageID }

// peerPair carries the sender and recipient for a single routed message.
// Bundling them prevents accidental argument swap at call sites.
type peerPair struct {
	from sharedmodel.Peer
	to   sharedmodel.Peer
}
