package handler

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
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

	gates, next, err := g.srv.ListGates(ctx, sharedmodel.ListFilter{
		Page: page,
		Size: size,
	})
	if err != nil {
		g.logger.Error("grpc: list gates failed", slog.Any("err", err))
		return nil, err
	}

	items := make([]*impb.ProviderSummary, len(gates))
	for i, v := range gates {
		appID := ""
		if v.ProviderAppID != nil {
			appID = *v.ProviderAppID
		}

		items[i] = &impb.ProviderSummary{
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
		}
	}

	return &impb.ProviderListGatesResponse{
		Items: items,
		Page:  int32(page),
		Size:  int32(len(items)),
		Next:  next,
	}, nil
}

// toProtoType maps internal domain GateType to Proto ProviderType enum.
func toProtoType(t sharedmodel.GateType) impb.ProviderType {
	switch t {
	case sharedmodel.TypeFacebook:
		return impb.ProviderType_PROVIDER_TYPE_FACEBOOK
	case sharedmodel.TypeInstagram:
		return impb.ProviderType_PROVIDER_TYPE_INSTAGRAM
	case sharedmodel.TypeWhatsApp:
		return impb.ProviderType_PROVIDER_TYPE_WHATSAPP
	case sharedmodel.TypeTelegramBot:
		return impb.ProviderType_PROVIDER_TYPE_TELEGRAM_BOT
	case sharedmodel.TypeTelegramApp:
		return impb.ProviderType_PROVIDER_TYPE_TELEGRAM_APP
	case sharedmodel.TypeViber:
		return impb.ProviderType_PROVIDER_TYPE_VIBER
	default:
		return impb.ProviderType_PROVIDER_TYPE_UNSPECIFIED
	}
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
