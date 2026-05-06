package facebook

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/provider/facebook/graph"
	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/im-providers-service/internal/store"
)

type facebookProvider struct {
	api         *Client
	logger      *slog.Logger
	messenger   service.Messenger
	gateCache   store.GateCache
	userCache   store.ExternalUserCache
	repo        store.FacebookStore
	metaAppRepo store.MetaAppStore
	gatewayer   *imgateway.Client
	media       service.MediaManager
	httpClient  *http.Client
}

func New(
	m service.Messenger,
	l *slog.Logger,
	gc store.GateCache,
	uc store.ExternalUserCache,
	repo store.FacebookStore,
	metaAppRepo store.MetaAppStore,
	gatewayer *imgateway.Client,
	media service.MediaManager,
) provider.Provider {
	return &facebookProvider{
		api:         NewClient(l),
		logger:      l.With("provider", "facebook"),
		messenger:   m,
		gateCache:   gc,
		userCache:   uc,
		repo:        repo,
		metaAppRepo: metaAppRepo,
		gatewayer:   gatewayer,
		media:       media,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *facebookProvider) Type() string { return "facebook" }

func (p *facebookProvider) Verify(ctx context.Context, query url.Values) (string, error) {
	req := graph.ParseVerify(query)
	if req.Mode != "subscribe" {
		return "", fmt.Errorf("unexpected hub.mode: %s", req.Mode)
	}

	uri := p.normalizeURI(ctx)
	app, err := p.metaAppRepo.SelectByURI(ctx, uri)
	if err != nil {
		return "", fmt.Errorf("verify: app lookup failed: %w", err)
	}

	if app.VerifyToken != "" && req.VerifyToken != app.VerifyToken {
		return "", fmt.Errorf("verify_token mismatch")
	}

	return req.Challenge, nil
}

func (p *facebookProvider) resolveGate(ctx context.Context, uri, pageID string) (*model.FacebookGate, error) {
	k := gateKey(uri, pageID)
	if cached, ok := p.gateCache.Get(k); ok && !cached.Enabled {
		return &model.FacebookGate{Enabled: false}, nil
	}
	g, err := p.repo.SelectByPageAndURI(ctx, pageID, uri)
	if err != nil {
		return nil, err
	}
	p.gateCache.Set(k, store.GateState{
		GateID: g.ID, Enabled: g.Enabled, Issuer: g.Peer.Iss, Sub: g.Peer.Sub, Domain: g.DomainID,
	})
	return g, nil
}

func (p *facebookProvider) normalizeURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string)
	if !strings.HasPrefix(uri, "/") {
		return "/" + uri
	}
	return uri
}

func (p *facebookProvider) fetchGate(ctx context.Context, gateID string) (*model.FacebookGate, error) {
	return p.repo.Select(ctx, gateID)
}

func gateKey(uri, pageID string) string { return uri + ":" + pageID }
