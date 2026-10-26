package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"github.com/webitel/im-providers-service/internal/provider"
)

type GateHandler struct {
	logger *slog.Logger
	srv    sharedsvc.GateManager
	// capabilities maps a channel type (provider.Type()) to the
	// delivery-status receipts its adapter declares.
	capabilities map[string]*impb.ProviderStatusCapabilities
	impb.UnimplementedGateServiceServer
}

func NewGateHandler(logger *slog.Logger, srv sharedsvc.GateManager, providers []provider.Provider) *GateHandler {
	return &GateHandler{
		logger:       logger,
		srv:          srv,
		capabilities: collectCapabilities(providers),
	}
}

// collectCapabilities snapshots the status capabilities declared by the
// registered provider adapters. Adapters without a CapabilityReporter are
// honest "no receipts" channels and stay absent from the map.
func collectCapabilities(providers []provider.Provider) map[string]*impb.ProviderStatusCapabilities {
	caps := make(map[string]*impb.ProviderStatusCapabilities, len(providers))

	for _, p := range providers {
		reporter, ok := p.(provider.CapabilityReporter)
		if !ok {
			continue
		}

		c := reporter.Capabilities()
		caps[p.Type()] = &impb.ProviderStatusCapabilities{
			Delivered: c.SupportsDelivered,
			Read:      c.SupportsRead,
			Failed:    c.SupportsFailed,
			Typing:    c.SupportsTyping,
		}
	}

	return caps
}

// ListGates maps domain results to the unified Proto response.
func (g *GateHandler) ListGates(ctx context.Context, req *impb.ProviderListGatesRequest) (*impb.ProviderListGatesResponse, error) {
	page := int(req.GetPage())

	size := int(req.GetSize())
	if size <= 0 {
		size = 20
	}

	domainID, err := resolveDomainID(ctx, req.GetDomainId())
	if err != nil {
		return nil, err
	}

	gates, next, err := g.srv.ListGates(ctx, sharedmodel.ListFilter{
		DomainID: domainID,
		Page:     page,
		Size:     size,
		Types:    fromProtoTypes(req.GetTypes()),
		Status:   fromProtoStatus(req.GetStatus()),
		Q:        req.GetQ(),
	})
	if err != nil {
		g.logger.Error("grpc: list gates failed", slog.Any("err", err))

		return nil, err
	}

	items := make([]*impb.ProviderSummary, len(gates))
	for i, v := range gates {
		items[i] = g.toProtoSummary(v)
	}

	return &impb.ProviderListGatesResponse{
		Items: items,
		Page:  int32(page),
		Size:  int32(len(items)),
		Next:  next,
	}, nil
}

// GetGate returns a single gate of the caller's domain.
func (g *GateHandler) GetGate(ctx context.Context, req *impb.GetGateRequest) (*impb.GetGateResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Reject before the store: a malformed id would otherwise fail the uuid
	// cast in Postgres and surface as Internal.
	if _, err := uuid.Parse(req.GetId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "id must be a UUID")
	}

	domainID, err := resolveDomainID(ctx, req.GetDomainId())
	if err != nil {
		return nil, err
	}

	gate, err := g.srv.GetGate(ctx, domainID, req.GetId())
	if errors.Is(err, sharedstore.ErrNotFound) {
		return nil, status.Errorf(codes.NotFound, "gate not found: %s", req.GetId())
	}

	if err != nil {
		g.logger.ErrorContext(ctx, "grpc: get gate failed", "gate_id", req.GetId(), "err", err)

		return nil, status.Error(codes.Internal, "get gate failed")
	}

	return &impb.GetGateResponse{Item: g.toProtoSummary(gate)}, nil
}

// resolveDomainID picks the domain to scope gate reads to: the caller's own
// domain for user calls, the request domain for service (mTLS) calls, which
// carry no user identity.
func resolveDomainID(ctx context.Context, requested int64) (int64, error) {
	if identity, ok := auth.GetIdentityFromContext(ctx); ok && identity.GetDomainID() > 0 {
		return identity.GetDomainID(), nil
	}

	if requested > 0 {
		return requested, nil
	}

	return 0, status.Error(codes.InvalidArgument, "domain_id is required")
}

func (g *GateHandler) toProtoSummary(v *sharedmodel.GateSummary) *impb.ProviderSummary {
	appID := ""
	if v.ProviderAppID != nil {
		appID = *v.ProviderAppID
	}

	return &impb.ProviderSummary{
		Id:            v.ID,
		Name:          v.Name,
		Type:          toProtoType(v.Type),
		Status:        toProtoStatus(v.Status),
		WebhookUrl:    v.WebhookURL,
		Contact:       v.Contact,
		ProviderAppId: appID,
		CreatedAt:     v.CreatedAt.UnixMilli(),
		UpdatedAt:     v.UpdatedAt.UnixMilli(),
		Capabilities:  g.capabilities[v.Type.String()],
		Address:       v.Address,
	}
}

// fromProtoTypes maps the requested provider types onto gate types, skipping
// the ones that have no gate counterpart.
func fromProtoTypes(types []impb.ProviderType) []sharedmodel.GateType {
	out := make([]sharedmodel.GateType, 0, len(types))
	for _, t := range types {
		for gt, pt := range gateTypes {
			if pt == t {
				out = append(out, gt)
			}
		}
	}

	return out
}

// fromProtoStatus maps the requested provider status onto a gate status;
// unspecified means "any".
func fromProtoStatus(s impb.ProviderStatus) sharedmodel.GateStatus {
	switch s {
	case impb.ProviderStatus_PROVIDER_STATUS_ACTIVE:
		return sharedmodel.StatusActive
	case impb.ProviderStatus_PROVIDER_STATUS_INACTIVE:
		return sharedmodel.StatusDisabled
	case impb.ProviderStatus_PROVIDER_STATUS_ERROR:
		return sharedmodel.StatusError
	case impb.ProviderStatus_PROVIDER_STATUS_UNSPECIFIED:
		return sharedmodel.StatusUnknown
	default:
		return sharedmodel.StatusUnknown
	}
}

// gateTypes maps internal gate types to their proto provider types.
var gateTypes = map[sharedmodel.GateType]impb.ProviderType{
	sharedmodel.TypeFacebook:    impb.ProviderType_PROVIDER_TYPE_FACEBOOK,
	sharedmodel.TypeInstagram:   impb.ProviderType_PROVIDER_TYPE_INSTAGRAM,
	sharedmodel.TypeWhatsApp:    impb.ProviderType_PROVIDER_TYPE_WHATSAPP,
	sharedmodel.TypeTelegramBot: impb.ProviderType_PROVIDER_TYPE_TELEGRAM_BOT,
	sharedmodel.TypeTelegramApp: impb.ProviderType_PROVIDER_TYPE_TELEGRAM_APP,
	sharedmodel.TypeViber:       impb.ProviderType_PROVIDER_TYPE_VIBER,
	sharedmodel.TypeCustom:      impb.ProviderType_PROVIDER_TYPE_CUSTOM,
	sharedmodel.TypeViberBM:     impb.ProviderType_PROVIDER_TYPE_VIBER_BM,
}

// toProtoType maps internal domain GateType to Proto ProviderType enum.
func toProtoType(t sharedmodel.GateType) impb.ProviderType {
	if pt, ok := gateTypes[t]; ok {
		return pt
	}

	return impb.ProviderType_PROVIDER_TYPE_UNSPECIFIED
}

// toProtoStatus maps internal domain GateStatus to Proto ProviderStatus enum.
func toProtoStatus(s sharedmodel.GateStatus) impb.ProviderStatus {
	switch s {
	case sharedmodel.StatusActive:
		return impb.ProviderStatus_PROVIDER_STATUS_ACTIVE
	case sharedmodel.StatusDisabled:
		return impb.ProviderStatus_PROVIDER_STATUS_INACTIVE
	case sharedmodel.StatusError:
		return impb.ProviderStatus_PROVIDER_STATUS_ERROR
	default:
		return impb.ProviderStatus_PROVIDER_STATUS_UNSPECIFIED
	}
}
