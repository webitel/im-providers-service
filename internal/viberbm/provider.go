// Package viberbm implements the InfoBip Viber Business Messages provider —
// distinct from the direct Viber Bot (internal/viber): traffic flows through a
// per-account InfoBip host, recipients are MSISDNs, inbound arrives as forwards.
// https://www.infobip.com/docs/api/channels/viber/viber-business-messages
package viberbm

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/internal/provider"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
	vibbmstore "github.com/webitel/im-providers-service/internal/viberbm/store"
)

type viberBMProvider struct {
	api           *apiClient
	logger        *slog.Logger
	messenger     sharedsvc.Messenger
	gateCache     sharedstore.GateCache
	userCache     sharedstore.ExternalUserCache
	repo          vibbmstore.ViberBMStore
	gatewayer     *imgateway.Client
	media         sharedsvc.MediaManager
	contactClient *imcontact.Client
	rdb           *redis.Client
	linkClient    *http.Client
	status        statusReporter
}

// statusReporter is the delivery-status surface the webhook pipeline needs;
// satisfied by core/service.StatusReporter.
type statusReporter interface {
	DeliveredByProviderIDs(ctx context.Context, gateID string, providerMessageIDs []string, at time.Time)
	ReadUpTo(ctx context.Context, gateID, providerUserID string, watermark time.Time)
	FailedByProviderID(ctx context.Context, gateID, providerMessageID string, at time.Time, errorCode, errorMessage string)
}

func New(
	m sharedsvc.Messenger,
	l *slog.Logger,
	gc sharedstore.GateCache,
	uc sharedstore.ExternalUserCache,
	repo vibbmstore.ViberBMStore,
	gatewayer *imgateway.Client,
	media sharedsvc.MediaManager,
	contactClient *imcontact.Client,
	rdb *redis.Client,
	api *apiClient,
	status *sharedsvc.StatusReporter,
) (*viberBMProvider, error) {
	return &viberBMProvider{
		api:           api,
		logger:        l.With("provider", "viber_bm"),
		messenger:     m,
		gateCache:     gc,
		userCache:     uc,
		repo:          repo,
		gatewayer:     gatewayer,
		media:         media,
		contactClient: contactClient,
		rdb:           rdb,
		linkClient:    newGuardedClient(30 * time.Second),
		status:        status,
	}, nil
}

var (
	_ provider.Provider           = (*viberBMProvider)(nil)
	_ provider.CapabilityReporter = (*viberBMProvider)(nil)
	_ provider.SignatureValidator = (*viberBMProvider)(nil)
)

func (p *viberBMProvider) Type() string { return "viber_bm" }

// Capabilities: InfoBip forwards DLR (delivered/failed) and Seen reports.
// Typing is not supported by the Viber BM channel.
func (p *viberBMProvider) Capabilities() sharedmodel.ProviderCapabilities {
	return sharedmodel.ProviderCapabilities{
		SupportsDelivered: true,
		SupportsRead:      true,
		SupportsFailed:    true,
		SupportsTyping:    false,
	}
}

// resolveGate returns the gate owning the webhook uri. The uri (unique per gate)
// identifies it — not `to`, which is the sender name only on MO and the customer
// MSISDN on DLR/Seen, so keying on it would drop every receipt. Disabled gates
// short-circuit from the LRU cache to skip a DB round-trip.
func (p *viberBMProvider) resolveGate(ctx context.Context, uri string) (*vibbmmodel.ViberBMGate, error) {
	if cached, ok := p.gateCache.Get(uri); ok && !cached.Enabled {
		return &vibbmmodel.ViberBMGate{Enabled: false}, nil
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

func (p *viberBMProvider) fetchGate(ctx context.Context, gateID string) (*vibbmmodel.ViberBMGate, error) {
	return p.repo.Select(ctx, gateID)
}

// webhookURI extracts the webhook path segment. Stored webhook_uri and the path
// param are slashless, so a stray leading slash is trimmed, not added.
func (p *viberBMProvider) webhookURI(ctx context.Context) string {
	uri, _ := ctx.Value(provider.WebhookURIKey).(string) //nolint:revive // missing key yields ""

	return strings.TrimPrefix(uri, "/")
}

type peerPair struct {
	from sharedmodel.Peer
	to   sharedmodel.Peer
}
