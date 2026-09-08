package handler

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customservice "github.com/webitel/im-providers-service/internal/custom/service"
)

type CustomHandler struct {
	impb.UnimplementedCustomServiceServer

	logger *slog.Logger
	srv    customservice.CustomManager
}

func NewCustomHandler(logger *slog.Logger, srv customservice.CustomManager) *CustomHandler {
	return &CustomHandler{logger: logger, srv: srv}
}

func (h *CustomHandler) CreateCustomGate(ctx context.Context, req *impb.ProviderCreateCustomGateRequest) (*impb.ProviderCreateCustomGateResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := h.srv.CreateGate(ctx, custommodel.CreateCustom{
		Name:             req.GetName(),
		Dc:               identity.GetDomainID(),
		Peer:             sharedmodel.Peer{Sub: req.GetPeer().GetSub(), Iss: req.GetPeer().GetIss()},
		CallbackURL:      req.GetCallbackUrl(),
		AppSecret:        req.GetAppSecret(),
		AllowedIPs:       req.GetAllowedIps(),
		RequestTimeoutMS: req.GetRequestTimeoutMs(),
		RetryAttempts:    req.GetRetryAttempts(),
	})
	if err != nil {
		return nil, toStatus(err, "create gate")
	}

	return &impb.ProviderCreateCustomGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *CustomHandler) GetCustomGate(ctx context.Context, req *impb.ProviderGetCustomGateRequest) (*impb.ProviderGetCustomGateResponse, error) {
	gate, err := h.srv.GetGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "get gate")
	}

	return &impb.ProviderGetCustomGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *CustomHandler) UpdateCustomGate(ctx context.Context, req *impb.ProviderUpdateCustomGateRequest) (*impb.ProviderUpdateCustomGateResponse, error) {
	name := req.GetName()
	callbackURL := req.GetCallbackUrl()
	appSecret := req.GetAppSecret()
	allowedIPs := req.GetAllowedIps()
	timeout := req.GetRequestTimeoutMs()
	retries := req.GetRetryAttempts()
	enabled := req.GetEnabled()

	upd := custommodel.UpdateCustom{
		ID:               req.GetId(),
		Name:             &name,
		CallbackURL:      &callbackURL,
		AppSecret:        &appSecret,
		AllowedIPs:       &allowedIPs,
		RequestTimeoutMS: &timeout,
		RetryAttempts:    &retries,
		Enabled:          &enabled,
	}
	if p := req.GetPeer(); p != nil {
		upd.Peer = &sharedmodel.Peer{Sub: p.GetSub(), Iss: p.GetIss()}
	}

	gate, err := h.srv.UpdateGate(ctx, upd)
	if err != nil {
		return nil, toStatus(err, "update gate")
	}

	return &impb.ProviderUpdateCustomGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *CustomHandler) DeleteCustomGate(ctx context.Context, req *impb.ProviderDeleteCustomGateRequest) (*impb.ProviderDeleteCustomGateResponse, error) {
	gate, err := h.srv.DeleteGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "delete gate")
	}

	return &impb.ProviderDeleteCustomGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *CustomHandler) gateToProto(g *custommodel.CustomGate) *impb.ProviderCustomGate {
	if g == nil {
		return nil
	}

	return &impb.ProviderCustomGate{
		Id:               g.ID,
		Name:             g.Name,
		WebhookUrl:       h.srv.WebhookURL(g.WebhookURI),
		CallbackUrl:      g.CallbackURL,
		AppSecret:        g.AppSecret,
		AllowedIps:       g.AllowedIPs,
		RequestTimeoutMs: g.RequestTimeoutMS,
		RetryAttempts:    g.RetryAttempts,
		Status:           toProtoStatus(g.Status),
		CreatedAt:        g.CreatedAt.UnixMilli(),
		UpdatedAt:        g.UpdatedAt.UnixMilli(),
		Enabled:          g.Enabled,
	}
}

func toProtoStatus(s sharedmodel.GateStatus) impb.ProviderStatus {
	switch s {
	case sharedmodel.StatusActive:
		return impb.ProviderStatus_PROVIDER_STATUS_ACTIVE
	case sharedmodel.StatusDisabled:
		return impb.ProviderStatus_PROVIDER_STATUS_INACTIVE
	case sharedmodel.StatusError:
		return impb.ProviderStatus_PROVIDER_STATUS_ERROR
	case sharedmodel.StatusUnknown:
		return impb.ProviderStatus_PROVIDER_STATUS_UNSPECIFIED
	default:
		return impb.ProviderStatus_PROVIDER_STATUS_UNSPECIFIED
	}
}
