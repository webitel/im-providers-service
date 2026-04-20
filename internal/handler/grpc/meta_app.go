package grpc

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ impb.MetaAppServiceServer = (*MetaAppHandler)(nil)

type MetaAppHandler struct {
	logger *slog.Logger
	srv    service.MetaAppManager
	impb.UnimplementedMetaAppServiceServer
}

func NewMetaAppHandler(logger *slog.Logger, srv service.MetaAppManager) *MetaAppHandler {
	return &MetaAppHandler{
		logger: logger,
		srv:    srv,
	}
}

// CreateMetaApp implements [provider.MetaAppServiceServer].
func (m *MetaAppHandler) CreateMetaApp(ctx context.Context, req *impb.ProviderCreateMetaAppRequest) (*impb.ProviderCreateMetaAppResponse, error) {
	app, err := m.srv.CreateMetaApp(ctx, model.CreateMetaApp{
		Name:             req.GetName(),
		AppID:            req.GetAppId(),
		AppSecret:        req.GetAppSecret(),
		OAuthRedirectURI: req.GetOauthRedirectUri(),
		Scopes:           req.GetScopes(),
	})
	if err != nil {
		m.logger.Error("failed to create meta app", slog.Any("err", err))
		return nil, status.Errorf(codes.Internal, "failed to create meta app: %v", err)
	}

	return &impb.ProviderCreateMetaAppResponse{
		Item: toProto(app),
	}, nil
}

// GetMetaApp implements [provider.MetaAppServiceServer].
func (m *MetaAppHandler) GetMetaApp(ctx context.Context, req *impb.ProviderGetMetaAppRequest) (*impb.ProviderGetMetaAppResponse, error) {
	app, err := m.srv.GetMetaApp(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "meta app not found: %v", err)
	}

	return &impb.ProviderGetMetaAppResponse{
		Item: toProto(app),
	}, nil
}

// UpdateMetaApp implements [provider.MetaAppServiceServer].
func (m *MetaAppHandler) UpdateMetaApp(ctx context.Context, req *impb.ProviderUpdateMetaAppRequest) (*impb.ProviderUpdateMetaAppResponse, error) {
	// Map proto request to domain Update model
	name := req.GetName()
	secret := req.GetAppSecret()
	uri := req.GetOauthRedirectUri()

	app, err := m.srv.UpdateMetaApp(ctx, model.UpdateMetaApp{
		ID:               req.GetId(),
		Name:             &name,
		AppSecret:        &secret,
		OAuthRedirectURI: &uri,
		Scopes:           req.GetScopes(),
	})
	if err != nil {
		m.logger.Error("failed to update meta app", slog.String("id", req.GetId()), slog.Any("err", err))
		return nil, status.Errorf(codes.Internal, "failed to update meta app: %v", err)
	}

	return &impb.ProviderUpdateMetaAppResponse{
		Item: toProto(app),
	}, nil
}

// DeleteMetaApp implements [provider.MetaAppServiceServer].
func (m *MetaAppHandler) DeleteMetaApp(ctx context.Context, req *impb.ProviderDeleteMetaAppRequest) (*impb.ProviderDeleteMetaAppResponse, error) {
	app, err := m.srv.DeleteMetaApp(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete meta app: %v", err)
	}

	return &impb.ProviderDeleteMetaAppResponse{
		Item: toProto(app),
	}, nil
}

// toProto helper to map domain model to protobuf message
func toProto(a *model.MetaApp) *impb.ProviderMetaApp {
	if a == nil {
		return nil
	}
	return &impb.ProviderMetaApp{
		Id:               a.ID,
		Name:             a.Name,
		AppId:            a.AppID,
		OauthRedirectUri: a.OAuthRedirectURI,
		Scopes:           a.Scopes,
		CreatedAt:        a.CreatedAt.UnixMilli(),
		UpdatedAt:        a.UpdatedAt.UnixMilli(),
	}
}
