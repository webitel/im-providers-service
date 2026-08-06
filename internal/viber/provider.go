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
	templates     *sharedsvc.TemplateRenderer
	receiverCache cache.Cache[string, string]
	httpClient    *http.Client
	linkClient    *http.Client
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
	templates *sharedsvc.TemplateRenderer,
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
		templates:     templates,
		receiverCache: receiverCache,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		linkClient:    newGuardedClient(30 * time.Second),
	}, nil
}

var _ provider.InteractiveSender = (*viberProvider)(nil)

func (p *viberProvider) Type() string { return "viber" }

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

func (p *viberProvider) webhookURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string)
	return uri
}

const maxSenderNameLen = 28

func senderOf(g *vibmodel.ViberGate, name ...string) sender {
	out := sender{Name: g.SenderName, Avatar: g.SenderAvatar}

	if len(name) > 0 && name[0] != "" {
		out.Name = trimSenderName(name[0])
		out.Avatar = ""
	}

	return out
}

func trimSenderName(name string) string {
	runes := []rune(name)
	if len(runes) <= maxSenderNameLen {
		return name
	}

	return string(runes[:maxSenderNameLen])
}

type peerPair struct {
	from sharedmodel.Peer
	to   sharedmodel.Peer
}
