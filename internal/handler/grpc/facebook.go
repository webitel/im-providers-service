package grpc

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/config"
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/infra/auth"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FacebookHandler implements gRPC service for Facebook gate management.
type FacebookHandler struct {
	logger *slog.Logger
	srv    service.FacebookManager
	cfg    *config.Config
	impb.UnimplementedFacebookServiceServer
}

// NewFacebookHandler creates a new gRPC handler instance.
func NewFacebookHandler(logger *slog.Logger, srv service.FacebookManager, cfg *config.Config) *FacebookHandler {
	return &FacebookHandler{logger: logger, srv: srv, cfg: cfg}
}

// CreateFacebookGate handles the creation of a new Facebook integration.
func (f *FacebookHandler) CreateFacebookGate(ctx context.Context, req *impb.ProviderCreateFacebookGateRequest) (*impb.ProviderCreateFacebookGateResponse, error) {
	auth, ok := auth.GetIdentityFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing identity in context")
	}

	gate, err := f.srv.CreateGate(ctx, model.CreateFacebook{
		Name:      req.GetName(),
		Dc:        auth.GetDomainID(),
		MetaAppID: req.GetMetaAppId(),
		PageID:    req.GetPageId(),
		PageToken: req.GetPageToken(),
		Peer:      model.Peer{Sub: req.GetPeer().Sub, Iss: req.GetPeer().Iss},
	})
	if err != nil {
		f.logger.Error("failed to create gate", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create gate: %v", err)
	}

	return &impb.ProviderCreateFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

// GetFacebookGate retrieves a single gate by its unique identifier.
func (f *FacebookHandler) GetFacebookGate(ctx context.Context, req *impb.ProviderGetFacebookGateRequest) (*impb.ProviderGetFacebookGateResponse, error) {
	gate, err := f.srv.GetGate(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gate not found")
	}
	return &impb.ProviderGetFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

// UpdateFacebookGate updates administrative settings for an existing gate.
func (f *FacebookHandler) UpdateFacebookGate(ctx context.Context, req *impb.ProviderUpdateFacebookGateRequest) (*impb.ProviderUpdateFacebookGateResponse, error) {
	name := req.GetName()
	enabled := req.GetEnabled()

	gate, err := f.srv.UpdateGate(ctx, model.UpdateFacebook{
		ID:      req.GetId(),
		Name:    &name,
		Enabled: &enabled,
		Peer:    &model.Peer{Sub: req.GetPeer().Sub, Iss: req.GetPeer().Iss},
	})
	if err != nil {
		f.logger.Error("failed to update gate", "id", req.GetId(), "error", err)
		return nil, status.Errorf(codes.Internal, "update failed: %v", err)
	}
	return &impb.ProviderUpdateFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

// DeleteFacebookGate removes the gate and its configuration.
func (f *FacebookHandler) DeleteFacebookGate(ctx context.Context, req *impb.ProviderDeleteFacebookGateRequest) (*impb.ProviderDeleteFacebookGateResponse, error) {
	gate, err := f.srv.DeleteGate(ctx, req.GetId())
	if err != nil {
		f.logger.Error("failed to delete gate", "id", req.GetId(), "error", err)
		return nil, status.Errorf(codes.Internal, "delete failed: %v", err)
	}
	return &impb.ProviderDeleteFacebookGateResponse{
		Item: f.gateToProto(gate),
	}, nil
}

// gateToProto converts the internal domain model to a gRPC message format.
func (f *FacebookHandler) gateToProto(g *model.FacebookGate) *impb.ProviderFacebookGate {
	if g == nil {
		return nil
	}

	// 2. Full mapping of the Facebook-specific gate fields.
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
