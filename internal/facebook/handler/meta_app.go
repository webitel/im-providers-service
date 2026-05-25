package handler

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbservice "github.com/webitel/im-providers-service/internal/facebook/service"
)

var _ impb.MetaAppServiceServer = (*MetaAppHandler)(nil)

type MetaAppHandler struct {
	logger *slog.Logger
	srv    fbservice.MetaAppManager
	impb.UnimplementedMetaAppServiceServer
}

func NewMetaAppHandler(logger *slog.Logger, srv fbservice.MetaAppManager) *MetaAppHandler {
	return &MetaAppHandler{logger: logger, srv: srv}
}

func (m *MetaAppHandler) CreateMetaApp(ctx context.Context, req *impb.ProviderCreateMetaAppRequest) (*impb.ProviderCreateMetaAppResponse, error) {
	app, err := m.srv.CreateMetaApp(ctx, fbmodel.CreateMetaApp{
		Name:             req.GetName(),
		URI:              req.GetUri(),
		AppID:            req.GetAppId(),
		AppSecret:        req.GetAppSecret(),
		OAuthRedirectURI: req.GetOauthRedirectUri(),
		Scopes:           req.GetScopes(),
		VerifyToken:      req.GetVerifyToken(),
	})
	if err != nil {
		return nil, toStatus(err, "create meta app")
	}

	return &impb.ProviderCreateMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

func (m *MetaAppHandler) GetMetaApp(ctx context.Context, req *impb.ProviderGetMetaAppRequest) (*impb.ProviderGetMetaAppResponse, error) {
	app, err := m.srv.GetMetaApp(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "get meta app")
	}

	return &impb.ProviderGetMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

func (m *MetaAppHandler) UpdateMetaApp(ctx context.Context, req *impb.ProviderUpdateMetaAppRequest) (*impb.ProviderUpdateMetaAppResponse, error) {
	name := req.GetName()
	secret := req.GetAppSecret()
	uri := req.GetOauthRedirectUri()
	verifyToken := req.GetVerifyToken()

	app, err := m.srv.UpdateMetaApp(ctx, fbmodel.UpdateMetaApp{
		ID:               req.GetId(),
		Name:             &name,
		AppSecret:        &secret,
		OAuthRedirectURI: &uri,
		Scopes:           req.GetScopes(),
		VerifyToken:      &verifyToken,
	})
	if err != nil {
		return nil, toStatus(err, "update meta app")
	}

	return &impb.ProviderUpdateMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

func (m *MetaAppHandler) DeleteMetaApp(ctx context.Context, req *impb.ProviderDeleteMetaAppRequest) (*impb.ProviderDeleteMetaAppResponse, error) {
	app, err := m.srv.DeleteMetaApp(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err, "delete meta app")
	}

	return &impb.ProviderDeleteMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

func metaAppToProto(a *fbmodel.MetaApp) *impb.ProviderMetaApp {
	if a == nil {
		return nil
	}
	return &impb.ProviderMetaApp{
		Id:               a.ID,
		Name:             a.Name,
		AppId:            a.AppID,
		OauthRedirectUri: a.OAuthRedirectURI,
		Scopes:           a.Scopes,
		VerifyToken:      a.VerifyToken,
		CreatedAt:        a.CreatedAt.UnixMilli(),
		UpdatedAt:        a.UpdatedAt.UnixMilli(),
	}
}
