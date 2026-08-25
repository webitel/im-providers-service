package handler

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/webitel/im-providers-service/config"
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	igservice "github.com/webitel/im-providers-service/internal/instagram/service"
)

type InstagramHandler struct {
	impb.UnimplementedInstagramServiceServer

	logger *slog.Logger
	srv    igservice.InstagramManager
	cfg    *config.Config
}

func NewInstagramHandler(logger *slog.Logger, srv igservice.InstagramManager, cfg *config.Config) *InstagramHandler {
	return &InstagramHandler{logger: logger, srv: srv, cfg: cfg}
}

func (i *InstagramHandler) CreateInstagramGate(ctx context.Context, req *impb.ProviderCreateInstagramGateRequest) (*impb.ProviderCreateInstagramGateResponse, error) {
	auth, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := i.srv.CreateGate(ctx, igmodel.CreateInstagram{
		Name:              req.GetName(),
		Dc:                auth.GetDomainID(),
		MetaAppID:         req.GetMetaAppId(),
		BusinessAccountID: req.GetBusinessAccountId(),
		IGToken:           req.GetBusinessAccountToken(),
		Peer:              sharedmodel.Peer{Sub: req.GetPeer().GetSub(), Iss: req.GetPeer().GetIss()},
	})
	if err != nil {
		return nil, toStatus(err, "create gate")
	}

	return &impb.ProviderCreateInstagramGateResponse{
		Item: i.gateToProto(gate),
	}, nil
}

func (i *InstagramHandler) GetInstagramGate(ctx context.Context, req *impb.ProviderGetInstagramGateRequest) (*impb.ProviderGetInstagramGateResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := i.srv.GetGate(ctx, req.GetId(), identity.GetDomainID())
	if err != nil {
		return nil, toStatus(err, "get gate")
	}

	return &impb.ProviderGetInstagramGateResponse{
		Item: i.gateToProto(gate),
	}, nil
}

func (i *InstagramHandler) UpdateInstagramGate(ctx context.Context, req *impb.ProviderUpdateInstagramGateRequest) (*impb.ProviderUpdateInstagramGateResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	// enabled is always applied (the proto has no field presence, so an omitted
	// value cannot be told apart from an explicit false — the toggle is required).
	enabled := req.GetEnabled()
	upd := igmodel.UpdateInstagram{
		ID:      req.GetId(),
		Dc:      identity.GetDomainID(),
		Enabled: &enabled,
	}

	// Only set name/peer when provided, so a partial update never blanks them.
	if name := req.GetName(); name != "" {
		upd.Name = &name
	}

	if p := req.GetPeer(); p != nil && (p.GetSub() != "" || p.GetIss() != "") {
		upd.Peer = &sharedmodel.Peer{Sub: p.GetSub(), Iss: p.GetIss()}
	}

	gate, err := i.srv.UpdateGate(ctx, upd)
	if err != nil {
		return nil, toStatus(err, "update gate")
	}

	return &impb.ProviderUpdateInstagramGateResponse{
		Item: i.gateToProto(gate),
	}, nil
}

func (i *InstagramHandler) DeleteInstagramGate(ctx context.Context, req *impb.ProviderDeleteInstagramGateRequest) (*impb.ProviderDeleteInstagramGateResponse, error) {
	identity, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := i.srv.DeleteGate(ctx, req.GetId(), identity.GetDomainID())
	if err != nil {
		return nil, toStatus(err, "delete gate")
	}

	return &impb.ProviderDeleteInstagramGateResponse{
		Item: i.gateToProto(gate),
	}, nil
}

// gateToProto maps the domain InstagramGate to the proto ProviderInstagramGate.
// Exposes business_account_id and name for WMSG-377 gateway-info resolution,
// but never exposes the access token.
func (i *InstagramHandler) gateToProto(g *igmodel.InstagramGate) *impb.ProviderInstagramGate {
	if g == nil {
		return nil
	}

	return &impb.ProviderInstagramGate{
		Id:                g.ID,
		Name:              g.Name,
		MetaAppId:         g.MetaAppID,
		BusinessAccountId: g.BusinessAccountID,
		Status:            impb.ProviderStatus(g.Status),
		CreatedAt:         g.CreatedAt.UnixMilli(),
		UpdatedAt:         g.UpdatedAt.UnixMilli(),
		Enabled:           g.Enabled,
	}
}
