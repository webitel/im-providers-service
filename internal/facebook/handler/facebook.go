package handler

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/config"
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbservice "github.com/webitel/im-providers-service/internal/facebook/service"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FacebookHandler struct {
	logger *slog.Logger
	srv    fbservice.FacebookManager
	cfg    *config.Config
	impb.UnimplementedFacebookServiceServer
}

func NewFacebookHandler(logger *slog.Logger, srv fbservice.FacebookManager, cfg *config.Config) *FacebookHandler {
	return &FacebookHandler{logger: logger, srv: srv, cfg: cfg}
}

func (f *FacebookHandler) CreateFacebookGate(ctx context.Context, req *impb.ProviderCreateFacebookGateRequest) (*impb.ProviderCreateFacebookGateResponse, error) {
	auth, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := f.srv.CreateGate(ctx, fbmodel.CreateFacebook{
		Name:      req.GetName(),
		Dc:        auth.GetDomainID(),
		MetaAppID: req.GetMetaAppId(),
		PageID:    req.GetPageId(),
		PageToken: req.GetPageToken(),
		Peer:      sharedmodel.Peer{Sub: req.GetPeer().Sub, Iss: req.GetPeer().Iss},
	})
	if err != nil {
		return nil, toStatus(err, "create gate")
	}

	return &impb.ProviderCreateFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

func (f *FacebookHandler) GetFacebookGate(ctx context.Context, req *impb.ProviderGetFacebookGateRequest) (*impb.ProviderGetFacebookGateResponse, error) {
	gate, err := f.srv.GetGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "get gate")
	}
	return &impb.ProviderGetFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

func (f *FacebookHandler) UpdateFacebookGate(ctx context.Context, req *impb.ProviderUpdateFacebookGateRequest) (*impb.ProviderUpdateFacebookGateResponse, error) {
	name := req.GetName()
	enabled := req.GetEnabled()

	gate, err := f.srv.UpdateGate(ctx, fbmodel.UpdateFacebook{
		ID:      req.GetId(),
		Name:    &name,
		Enabled: &enabled,
		Peer:    &sharedmodel.Peer{Sub: req.GetPeer().Sub, Iss: req.GetPeer().Iss},
	})
	if err != nil {
		return nil, toStatus(err, "update gate")
	}
	return &impb.ProviderUpdateFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

func (f *FacebookHandler) DeleteFacebookGate(ctx context.Context, req *impb.ProviderDeleteFacebookGateRequest) (*impb.ProviderDeleteFacebookGateResponse, error) {
	gate, err := f.srv.DeleteGate(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "delete gate")
	}
	return &impb.ProviderDeleteFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

func (f *FacebookHandler) gateToProto(g *fbmodel.FacebookGate) *impb.ProviderFacebookGate {
	if g == nil {
		return nil
	}
	return &impb.ProviderFacebookGate{
		Id:        g.ID,
		Name:      g.Name,
		MetaAppId: g.MetaAppID,
		PageId:    g.PageID,
		PageName:  g.PageName,
		Status:    impb.ProviderStatus(g.Status),
		CreatedAt: g.CreatedAt.UnixMilli(),
		UpdatedAt: g.UpdatedAt.UnixMilli(),
		Enabled:   g.Enabled,
	}
}
