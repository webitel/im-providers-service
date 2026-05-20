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

// --- FacebookHandler ---

// FacebookHandler implements gRPC service for Facebook gate management.
type FacebookHandler struct {
	logger *slog.Logger
	srv    fbservice.FacebookManager
	cfg    *config.Config
	impb.UnimplementedFacebookServiceServer
}

// NewFacebookHandler creates a new gRPC handler instance.
func NewFacebookHandler(logger *slog.Logger, srv fbservice.FacebookManager, cfg *config.Config) *FacebookHandler {
	return &FacebookHandler{logger: logger, srv: srv, cfg: cfg}
}

// CreateFacebookGate handles the creation of a new Facebook integration.
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

	gate, err := f.srv.UpdateGate(ctx, fbmodel.UpdateFacebook{
		ID:      req.GetId(),
		Name:    &name,
		Enabled: &enabled,
		Peer:    &sharedmodel.Peer{Sub: req.GetPeer().Sub, Iss: req.GetPeer().Iss},
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

// --- MetaAppHandler ---

var _ impb.MetaAppServiceServer = (*MetaAppHandler)(nil)

type MetaAppHandler struct {
	logger *slog.Logger
	srv    fbservice.MetaAppManager
	impb.UnimplementedMetaAppServiceServer
}

func NewMetaAppHandler(logger *slog.Logger, srv fbservice.MetaAppManager) *MetaAppHandler {
	return &MetaAppHandler{
		logger: logger,
		srv:    srv,
	}
}

// CreateMetaApp implements [provider.MetaAppServiceServer].
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
		m.logger.Error("failed to create meta app", slog.Any("err", err))
		return nil, status.Errorf(codes.Internal, "failed to create meta app: %v", err)
	}

	return &impb.ProviderCreateMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

// GetMetaApp implements [provider.MetaAppServiceServer].
func (m *MetaAppHandler) GetMetaApp(ctx context.Context, req *impb.ProviderGetMetaAppRequest) (*impb.ProviderGetMetaAppResponse, error) {
	app, err := m.srv.GetMetaApp(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "meta app not found: %v", err)
	}

	return &impb.ProviderGetMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

// UpdateMetaApp implements [provider.MetaAppServiceServer].
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
		m.logger.Error("failed to update meta app", slog.String("id", req.GetId()), slog.Any("err", err))
		return nil, status.Errorf(codes.Internal, "failed to update meta app: %v", err)
	}

	return &impb.ProviderUpdateMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

// DeleteMetaApp implements [provider.MetaAppServiceServer].
func (m *MetaAppHandler) DeleteMetaApp(ctx context.Context, req *impb.ProviderDeleteMetaAppRequest) (*impb.ProviderDeleteMetaAppResponse, error) {
	app, err := m.srv.DeleteMetaApp(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete meta app: %v", err)
	}

	return &impb.ProviderDeleteMetaAppResponse{
		Item: metaAppToProto(app),
	}, nil
}

// metaAppToProto maps domain model to protobuf message.
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

// --- MetaOauthHandler ---

var _ impb.MetaOAuthServiceServer = (*MetaOauthHandler)(nil)

type MetaOauthHandler struct {
	logger *slog.Logger
	srv    fbservice.MetaOAuthManager
	impb.UnimplementedMetaOAuthServiceServer
}

func NewMetaOauthHandler(logger *slog.Logger, srv fbservice.MetaOAuthManager) *MetaOauthHandler {
	return &MetaOauthHandler{
		logger: logger,
		srv:    srv,
	}
}

// StartMetaOAuth initiates the OAuth flow by providing a redirect URL.
func (h *MetaOauthHandler) StartMetaOAuth(ctx context.Context, req *impb.ProviderMetaOAuthStartRequest) (*impb.ProviderMetaOAuthStartResponse, error) {
	authURL, state, err := h.srv.StartOAuth(ctx, fbmodel.OAuthStart{
		MetaAppID: req.GetMetaAppId(),
	})
	if err != nil {
		h.logger.Error("failed to initiate oauth", slog.Any("err", err))
		return nil, status.Errorf(codes.Internal, "failed to start oauth flow: %v", err)
	}

	return &impb.ProviderMetaOAuthStartResponse{
		AuthUrl: authURL,
		State:   state,
	}, nil
}

// MetaOAuthCallback handles the code-to-token exchange and returns discovered pages.
func (h *MetaOauthHandler) MetaOAuthCallback(ctx context.Context, req *impb.ProviderMetaOAuthCallbackRequest) (*impb.ProviderMetaOAuthCallbackResponse, error) {
	userToken, pages, err := h.srv.HandleCallback(ctx, fbmodel.OAuthCallback{
		MetaAppID: req.GetMetaAppId(),
		Code:      req.GetCode(),
		State:     req.GetState(),
	})
	if err != nil {
		h.logger.Error("oauth callback exchange failed", slog.Any("err", err))
		return nil, status.Errorf(codes.Unauthenticated, "failed to exchange authorization code: %v", err)
	}

	linkedPages := make([]*impb.ProviderMetaLinkedPage, len(pages))
	for i, p := range pages {
		linkedPages[i] = &impb.ProviderMetaLinkedPage{
			PageId:      p.PageID,
			PageName:    p.Name,
			AccessToken: p.PageToken,
			Platform:    "facebook",
		}
	}

	return &impb.ProviderMetaOAuthCallbackResponse{
		UserAccessToken: userToken,
		Pages:           linkedPages,
	}, nil
}
