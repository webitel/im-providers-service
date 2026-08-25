// Package instagram implements the Instagram Direct Messaging provider.
// https://developers.instagram.com/docs/instagram-api/messaging
package instagram

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/webitel/webitel-go-kit/pkg/cache"

	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	igstore "github.com/webitel/im-providers-service/internal/instagram/store"
	"github.com/webitel/im-providers-service/internal/provider"
)

type instagramProvider struct {
	api           graphAPI
	logger        *slog.Logger
	messenger     sharedsvc.Messenger
	gateCache     sharedstore.GateCache
	userCache     sharedstore.ExternalUserCache
	repo          igstore.InstagramStore
	metaAppRepo   fbstore.MetaAppStore
	gatewayer     *imgateway.Client
	media         sharedsvc.MediaManager
	contactClient *imcontact.Client
	rdb           *redis.Client
	// igsidCache maps internal contact UUID → Instagram IGSID to avoid an
	// im-contact round-trip on every outbound message.
	igsidCache cache.Cache[string, string]
	// httpClient is used exclusively for media downloads; kept separate from
	// api.http so the two timeouts can be tuned independently.
	httpClient *http.Client
	// status reports delivery/read/failed receipts back to im-thread-service.
	status statusReporter
	// templates renders system event messages; nil means feature is disabled.
	templates *sharedsvc.TemplateRenderer
}

// statusReporter is the delivery-status surface the webhook pipeline needs;
// satisfied by core/service.StatusReporter. Narrowed to an interface so
// receipt routing is testable without the thread client.
type statusReporter interface {
	DeliveredByProviderIDs(ctx context.Context, gateID string, providerMessageIDs []string, at time.Time)
	DeliveredUpTo(ctx context.Context, gateID, providerUserID string, watermark time.Time)
	ReadUpTo(ctx context.Context, gateID, providerUserID string, watermark time.Time)
}

func New(
	m sharedsvc.Messenger,
	l *slog.Logger,
	gc sharedstore.GateCache,
	uc sharedstore.ExternalUserCache,
	repo igstore.InstagramStore,
	metaAppRepo fbstore.MetaAppStore,
	gatewayer *imgateway.Client,
	media sharedsvc.MediaManager,
	contactClient *imcontact.Client,
	rdb *redis.Client,
	api *apiClient,
	status *sharedsvc.StatusReporter,
	templates *sharedsvc.TemplateRenderer,
) (provider.Provider, error) {
	igsidCache, err := cache.New[string, string]().
		L1(cache.RistrettoConfig{MaxCost: 1000, NumCounters: 10000}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("instagram provider: init igsid cache: %w", err)
	}

	return &instagramProvider{
		api:           NewResilientGraphAPI(api, l),
		logger:        l.With("provider", "instagram"),
		messenger:     m,
		gateCache:     gc,
		userCache:     uc,
		repo:          repo,
		metaAppRepo:   metaAppRepo,
		gatewayer:     gatewayer,
		media:         media,
		contactClient: contactClient,
		rdb:           rdb,
		igsidCache:    igsidCache,
		httpClient:    newGuardedClient(30 * time.Second),
		status:        status,
		templates:     templates,
	}, nil
}

var (
	_ provider.Sender             = (*instagramProvider)(nil)
	_ provider.Receiver           = (*instagramProvider)(nil)
	_ provider.Verifier           = (*instagramProvider)(nil)
	_ provider.SignatureValidator = (*instagramProvider)(nil)
	_ provider.InteractiveSender  = (*instagramProvider)(nil)
	_ provider.CapabilityReporter = (*instagramProvider)(nil)
	_ provider.TypingSender       = (*instagramProvider)(nil)
)

func (p *instagramProvider) Type() string {
	return sharedmodel.TypeInstagram.String()
}

// Capabilities: Instagram emits message_deliveries and message_reads
// webhooks; failures surface as synchronous Send API errors.
func (p *instagramProvider) Capabilities() sharedmodel.ProviderCapabilities {
	return sharedmodel.ProviderCapabilities{
		SupportsDelivered: true,
		SupportsRead:      true,
		SupportsFailed:    true,
		SupportsTyping:    true,
	}
}

func (p *instagramProvider) Verify(ctx context.Context, query url.Values) (string, error) {
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
		return "", fmt.Errorf("verify_token mismatch") //nolint:revive // false positive, wrapped error needed for context
	}

	return req.Challenge, nil
}

// resolveGate returns the InstagramGate for the given business account. Disabled gates are
// short-circuited from the LRU cache to avoid an unnecessary DB round-trip on
// every webhook delivery.
func (p *instagramProvider) resolveGate(ctx context.Context, uri, businessAccountID string) (*igmodel.InstagramGate, error) { //nolint:unused // will be called in webhook handlers
	k := gateKey(businessAccountID)
	if cached, ok := p.gateCache.Get(k); ok && !cached.Enabled {
		return &igmodel.InstagramGate{Enabled: false}, nil
	}

	g, err := p.repo.SelectByBusinessAccountAndURI(ctx, businessAccountID, uri)
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

func (p *instagramProvider) fetchGate(ctx context.Context, gateID string) (*igmodel.InstagramGate, error) { //nolint:unused // will be called in webhook handlers
	return p.repo.Select(ctx, gateID)
}

// webhookURI extracts and normalises the webhook path segment from context.
func (p *instagramProvider) webhookURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string) //nolint:revive // type assertion ignored as per Facebook pattern
	if !strings.HasPrefix(uri, "/") {
		return "/" + uri
	}

	return uri
}

func gateKey(businessAccountID string) string {
	return igstore.GateCacheKey(businessAccountID)
}

// peerPair carries the sender and recipient for a single routed message.
// Bundling them prevents accidental argument swap at call sites.
type peerPair struct { //nolint:unused // will be used in webhook handling
	from sharedmodel.Peer
	to   sharedmodel.Peer
}

func (p *instagramProvider) SendTyping(ctx context.Context, req *provider.TypingRequest) error {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return err
	}

	igsid, err := p.resolveIGSID(ctx, g, req.ExternalID)
	if err != nil {
		return err
	}

	return p.api.SendTyping(ctx, g.IGToken, igsid, req.TypingOn)
}
