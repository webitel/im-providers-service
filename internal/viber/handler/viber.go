package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/im-providers-service/config"
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
	vibservice "github.com/webitel/im-providers-service/internal/viber/service"
)

type ViberHandler struct {
	logger *slog.Logger
	srv    vibservice.ViberManager
	cfg    *config.Config
	impb.UnimplementedViberServiceServer
}

func NewViberHandler(logger *slog.Logger, srv vibservice.ViberManager, cfg *config.Config) *ViberHandler {
	return &ViberHandler{logger: logger, srv: srv, cfg: cfg}
}

func (h *ViberHandler) CreateViberGate(ctx context.Context, req *impb.ProviderCreateViberGateRequest) (*impb.ProviderCreateViberGateResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := h.srv.CreateGate(ctx, vibmodel.CreateViber{
		Name:         req.GetName(),
		Dc:           identity.GetDomainID(),
		Peer:         sharedmodel.Peer{Sub: req.GetPeer().GetSub(), Iss: req.GetPeer().GetIss()},
		AuthToken:    req.GetAuthToken(),
		SenderName:   req.GetSenderName(),
		SenderAvatar: req.GetSenderAvatar(),
	})
	if err != nil {
		return nil, toStatus(err, "create gate")
	}

	return &impb.ProviderCreateViberGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberHandler) GetViberGate(ctx context.Context, req *impb.ProviderGetViberGateRequest) (*impb.ProviderGetViberGateResponse, error) {
	gate, err := h.srv.GetGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "get gate")
	}
	return &impb.ProviderGetViberGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberHandler) UpdateViberGate(ctx context.Context, req *impb.ProviderUpdateViberGateRequest) (*impb.ProviderUpdateViberGateResponse, error) {
	name := req.GetName()
	authToken := req.GetAuthToken()
	senderName := req.GetSenderName()
	senderAvatar := req.GetSenderAvatar()
	enabled := req.GetEnabled()

	upd := vibmodel.UpdateViber{
		ID:           req.GetId(),
		Name:         &name,
		AuthToken:    &authToken,
		SenderName:   &senderName,
		SenderAvatar: &senderAvatar,
		Enabled:      &enabled,
	}
	if p := req.GetPeer(); p != nil {
		upd.Peer = &sharedmodel.Peer{Sub: p.GetSub(), Iss: p.GetIss()}
	}

	gate, err := h.srv.UpdateGate(ctx, upd)
	if err != nil {
		return nil, toStatus(err, "update gate")
	}
	return &impb.ProviderUpdateViberGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberHandler) DeleteViberGate(ctx context.Context, req *impb.ProviderDeleteViberGateRequest) (*impb.ProviderDeleteViberGateResponse, error) {
	gate, err := h.srv.DeleteGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "delete gate")
	}
	return &impb.ProviderDeleteViberGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberHandler) gateToProto(g *vibmodel.ViberGate) *impb.ProviderViberGate {
	if g == nil {
		return nil
	}
	return &impb.ProviderViberGate{
		Id:           g.ID,
		Name:         g.Name,
		BotId:        g.BotID,
		BotUri:       g.BotURI,
		SenderName:   g.SenderName,
		SenderAvatar: g.SenderAvatar,
		WebhookUrl:   h.webhookURL(g.WebhookURI),
		Status:       impb.ProviderStatus(g.Status),
		CreatedAt:    g.CreatedAt.UnixMilli(),
		UpdatedAt:    g.UpdatedAt.UnixMilli(),
		Enabled:      g.Enabled,
	}
}

func (h *ViberHandler) webhookURL(uri string) string {
	if uri == "" {
		return ""
	}
	base := strings.TrimRight(h.cfg.Service.PublicURL, "/")
	path := "/" + strings.Trim(h.cfg.Service.WebhookPath, "/")
	return fmt.Sprintf("%s%s/viber/%s", base, path, uri)
}
