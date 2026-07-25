package viber

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/webitel-go-kit/pkg/cache"

	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/internal/provider"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
	vibstore "github.com/webitel/im-providers-service/internal/viber/store"
)

type viberProvider struct {
	api           *client
	logger        *slog.Logger
	messenger     sharedsvc.Messenger
	gateCache     sharedstore.GateCache
	userCache     sharedstore.ExternalUserCache
	repo          vibstore.ViberStore
	gatewayer     *imgateway.Client
	media         sharedsvc.MediaManager
	contactClient *imcontact.Client
	rdb           *redis.Client
	// receiverCache maps internal contact UUID → Viber user id to avoid an
	// im-contact round-trip on every outbound message.
	receiverCache cache.Cache[string, string]
	// httpClient is used exclusively for inbound media downloads.
	httpClient *http.Client
}

func New(
	m sharedsvc.Messenger,
	l *slog.Logger,
	gc sharedstore.GateCache,
	uc sharedstore.ExternalUserCache,
	repo vibstore.ViberStore,
	gatewayer *imgateway.Client,
	media sharedsvc.MediaManager,
	contactClient *imcontact.Client,
	rdb *redis.Client,
	api *client,
) (provider.Provider, error) {
	receiverCache, err := cache.New[string, string]().
		L1(cache.RistrettoConfig{MaxCost: 1000, NumCounters: 10000}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("viber provider: init receiver cache: %w", err)
	}
	return &viberProvider{
		api:           api,
		logger:        l.With("provider", "viber"),
		messenger:     m,
		gateCache:     gc,
		userCache:     uc,
		repo:          repo,
		gatewayer:     gatewayer,
		media:         media,
		contactClient: contactClient,
		rdb:           rdb,
		receiverCache: receiverCache,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

var _ provider.InteractiveSender = (*viberProvider)(nil)

func (p *viberProvider) Type() string { return "viber" }

// resolveGate resolves the gate by its secret webhook path segment. Disabled gates
// are short-circuited from the LRU cache to avoid a DB round-trip on every delivery.
func (p *viberProvider) resolveGate(ctx context.Context, uri string) (*vibmodel.ViberGate, error) {
	if cached, ok := p.gateCache.Get(uri); ok && !cached.Enabled {
		return &vibmodel.ViberGate{Enabled: false}, nil
	}

	g, err := p.repo.SelectByURI(ctx, uri)
	if err != nil {
		return nil, err
	}
	p.gateCache.Set(uri, sharedstore.GateState{
		GateID:  g.ID,
		Enabled: g.Enabled,
		Issuer:  g.Peer.Iss,
		Sub:     g.Peer.Sub,
		Domain:  g.DomainID,
	})
	return g, nil
}

func (p *viberProvider) fetchGate(ctx context.Context, gateID string) (*vibmodel.ViberGate, error) {
	return p.repo.Select(ctx, gateID)
}

// webhookURI extracts the webhook path segment injected by the HTTP layer. It is the
// unmodified secret token stored as gate_viber.webhook_uri.
func (p *viberProvider) webhookURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string)
	return uri
}

// senderOf builds the mandatory Viber sender block for outbound messages.
func senderOf(g *vibmodel.ViberGate) sender {
	return sender{Name: g.SenderName, Avatar: g.SenderAvatar}
}

// peerPair carries the sender and recipient for a single routed message.
type peerPair struct {
	from sharedmodel.Peer
	to   sharedmodel.Peer
}
