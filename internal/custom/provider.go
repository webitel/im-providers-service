package custom

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/webitel/webitel-go-kit/pkg/cache"

	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
	"github.com/webitel/im-providers-service/internal/provider"
)

type customProvider struct {
	api           *client
	fetch         *fetcher
	retries       *retryQueue
	logger        *slog.Logger
	messenger     sharedsvc.Messenger
	gateCache     sharedstore.GateCache
	userCache     sharedstore.ExternalUserCache
	repo          customstore.CustomStore
	gatewayer     *imgateway.Client
	media         sharedsvc.MediaManager
	contactClient *imcontact.Client
	rdb           *redis.Client
	templates     *sharedsvc.TemplateRenderer
	status        *sharedsvc.StatusReporter
	receiverCache cache.Cache[string, string]
}

func New(
	m sharedsvc.Messenger,
	l *slog.Logger,
	gc sharedstore.GateCache,
	uc sharedstore.ExternalUserCache,
	repo customstore.CustomStore,
	gatewayer *imgateway.Client,
	media sharedsvc.MediaManager,
	contactClient *imcontact.Client,
	rdb *redis.Client,
	api *client,
	fetch *fetcher,
	retries *retryQueue,
	templates *sharedsvc.TemplateRenderer,
	status *sharedsvc.StatusReporter,
) (provider.Provider, error) {
	receiverCache, err := cache.New[string, string]().
		L1(cache.RistrettoConfig{MaxCost: 1000, NumCounters: 10000}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("custom provider: init receiver cache: %w", err)
	}

	return &customProvider{
		api:           api,
		fetch:         fetch,
		retries:       retries,
		logger:        l.With("provider", custommodel.ProviderType),
		messenger:     m,
		gateCache:     gc,
		userCache:     uc,
		repo:          repo,
		gatewayer:     gatewayer,
		media:         media,
		contactClient: contactClient,
		rdb:           rdb,
		templates:     templates,
		status:        status,
		receiverCache: receiverCache,
	}, nil
}

var (
	_ provider.InteractiveSender  = (*customProvider)(nil)
	_ provider.LocationSender     = (*customProvider)(nil)
	_ provider.ContactSender      = (*customProvider)(nil)
	_ provider.SignatureValidator = (*customProvider)(nil)
	_ provider.WebhookResponder   = (*customProvider)(nil)
	_ provider.CapabilityReporter = (*customProvider)(nil)
)

func (p *customProvider) Type() string { return custommodel.ProviderType }

func (p *customProvider) Capabilities() sharedmodel.ProviderCapabilities {
	return sharedmodel.ProviderCapabilities{
		SupportsDelivered: true,
		SupportsRead:      true,
		SupportsFailed:    true,
	}
}

func (p *customProvider) resolveGate(ctx context.Context, uri string) (*custommodel.CustomGate, error) {
	if cached, ok := p.gateCache.Get(uri); ok && !cached.Enabled {
		return &custommodel.CustomGate{Enabled: false}, nil
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

func (p *customProvider) fetchGate(ctx context.Context, gateID string) (*custommodel.CustomGate, error) {
	return p.repo.Select(ctx, gateID)
}

func (p *customProvider) webhookURI(ctx context.Context) string {
	uri, ok := ctx.Value(provider.WebhookURIKey).(string)
	if !ok {
		return ""
	}

	return uri
}

type peerPair struct {
	from sharedmodel.Peer
	to   sharedmodel.Peer
}
