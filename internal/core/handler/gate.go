package handler

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
)

type GateHandler struct {
	logger *slog.Logger
	srv    sharedsvc.GateManager
	impb.UnimplementedGateServiceServer
}

func NewGateHandler(logger *slog.Logger, srv sharedsvc.GateManager) *GateHandler {
	return &GateHandler{logger: logger, srv: srv}
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
