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
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
	vibbmservice "github.com/webitel/im-providers-service/internal/viberbm/service"
)

type ViberBMHandler struct {
	impb.UnimplementedViberBmServiceServer

	logger *slog.Logger
	srv    vibbmservice.ViberBMManager
	cfg    *config.Config
}

func NewViberBMHandler(logger *slog.Logger, srv vibbmservice.ViberBMManager, cfg *config.Config) *ViberBMHandler {
	return &ViberBMHandler{logger: logger, srv: srv, cfg: cfg}
}

func (h *ViberBMHandler) CreateViberBm(ctx context.Context, req *impb.ProviderCreateViberBmGateRequest) (*impb.ProviderCreateViberBmGateResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := h.srv.CreateGate(ctx, vibbmmodel.CreateViberBM{
		Name:          req.GetName(),
		Dc:            identity.GetDomainID(),
		Peer:          sharedmodel.Peer{Sub: req.GetPeer().GetSub(), Iss: req.GetPeer().GetIss()},
		BaseURL:       req.GetBaseUrl(),
		SenderName:    req.GetSenderName(),
		APIKey:        req.GetApiKey(),
		WebhookSecret: req.GetWebhookSecret(),
	})
	if err != nil {
		return nil, toStatus(err, "create gate")
	}

	return &impb.ProviderCreateViberBmGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberBMHandler) GetViberBm(ctx context.Context, req *impb.ProviderGetViberBmGateRequest) (*impb.ProviderGetViberBmGateResponse, error) {
	gate, err := h.srv.GetGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "get gate")
	}

	return &impb.ProviderGetViberBmGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberBMHandler) UpdateViberBm(ctx context.Context, req *impb.ProviderUpdateViberBmGateRequest) (*impb.ProviderUpdateViberBmGateResponse, error) {
	name := req.GetName()
	baseURL := req.GetBaseUrl()
	senderName := req.GetSenderName()
	apiKey := req.GetApiKey()
	webhookSecret := req.GetWebhookSecret()
	enabled := req.GetEnabled()

	upd := vibbmmodel.UpdateViberBM{
		ID:            req.GetId(),
		Name:          &name,
		BaseURL:       &baseURL,
		SenderName:    &senderName,
		APIKey:        &apiKey,
		WebhookSecret: &webhookSecret,
		Enabled:       &enabled,
	}
	if p := req.GetPeer(); p != nil {
		upd.Peer = &sharedmodel.Peer{Sub: p.GetSub(), Iss: p.GetIss()}
	}

	gate, err := h.srv.UpdateGate(ctx, upd)
	if err != nil {
		return nil, toStatus(err, "update gate")
	}

	return &impb.ProviderUpdateViberBmGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberBMHandler) DeleteViberBm(ctx context.Context, req *impb.ProviderDeleteViberBmGateRequest) (*impb.ProviderDeleteViberBmGateResponse, error) {
	gate, err := h.srv.DeleteGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "delete gate")
	}

	return &impb.ProviderDeleteViberBmGateResponse{Item: h.gateToProto(gate)}, nil
}

func (h *ViberBMHandler) ListViberBm(ctx context.Context, req *impb.ProviderListViberBmGatesRequest) (*impb.ProviderListViberBmGatesResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gates, next, err := h.srv.ListGates(ctx, identity.GetDomainID(), req.GetQ(), int(req.GetPage()), int(req.GetSize()))
	if err != nil {
		return nil, toStatus(err, "list gates")
	}

	items := make([]*impb.ProviderViberBmGate, 0, len(gates))
	for _, g := range gates {
		items = append(items, h.gateToProto(g))
	}

	return &impb.ProviderListViberBmGatesResponse{Items: items, Next: next}, nil
}

func (h *ViberBMHandler) SendViberBmTemplate(ctx context.Context, req *impb.SendViberBmTemplateRequest) (*impb.SendViberBmTemplateResponse, error) {
	res, err := h.srv.SendTemplate(ctx, req.GetGateId(), req.GetTo(), req.GetTemplateId(), req.GetLanguage(), req.GetParameters())
	if err != nil {
		return nil, toStatus(err, "send template")
	}

	resp := &impb.SendViberBmTemplateResponse{MessageId: res.ID}
	if bulk, ok := res.MD["bulk_id"].(string); ok {
		resp.BulkId = bulk
	}

	return resp, nil
}

func (h *ViberBMHandler) gateToProto(g *vibbmmodel.ViberBMGate) *impb.ProviderViberBmGate {
	if g == nil {
		return nil
	}

	return &impb.ProviderViberBmGate{
		Id:         g.ID,
		Name:       g.Name,
		BaseUrl:    g.BaseURL,
		SenderName: g.SenderName,
		WebhookUrl: h.webhookURL(g.WebhookURI),
		Status:     impb.ProviderStatus(g.Status),
		CreatedAt:  g.CreatedAt.UnixMilli(),
		UpdatedAt:  g.UpdatedAt.UnixMilli(),
		Enabled:    g.Enabled,
	}
}

func (h *ViberBMHandler) webhookURL(uri string) string {
	if uri == "" {
		return ""
	}

	base := strings.TrimRight(h.cfg.Service.PublicURL, "/")
	path := "/" + strings.Trim(h.cfg.Service.WebhookPath, "/")

	return fmt.Sprintf("%s%s/viber_bm/%s", base, path, uri)
}
