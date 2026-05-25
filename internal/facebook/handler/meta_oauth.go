package handler

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbservice "github.com/webitel/im-providers-service/internal/facebook/service"
)

var _ impb.MetaOAuthServiceServer = (*MetaOauthHandler)(nil)

type MetaOauthHandler struct {
	logger *slog.Logger
	srv    fbservice.MetaOAuthManager
	impb.UnimplementedMetaOAuthServiceServer
}

func NewMetaOauthHandler(logger *slog.Logger, srv fbservice.MetaOAuthManager) *MetaOauthHandler {
	return &MetaOauthHandler{logger: logger, srv: srv}
}

func (h *MetaOauthHandler) StartMetaOAuth(ctx context.Context, req *impb.ProviderMetaOAuthStartRequest) (*impb.ProviderMetaOAuthStartResponse, error) {
	authURL, state, err := h.srv.StartOAuth(ctx, fbmodel.OAuthStart{
		MetaAppID: req.GetMetaAppId(),
	})
	if err != nil {
		return nil, toStatus(err, "start oauth")
	}

	return &impb.ProviderMetaOAuthStartResponse{
		AuthUrl: authURL,
		State:   state,
	}, nil
}

func (h *MetaOauthHandler) MetaOAuthCallback(ctx context.Context, req *impb.ProviderMetaOAuthCallbackRequest) (*impb.ProviderMetaOAuthCallbackResponse, error) {
	userToken, pages, err := h.srv.HandleCallback(ctx, fbmodel.OAuthCallback{
		MetaAppID: req.GetMetaAppId(),
		Code:      req.GetCode(),
		State:     req.GetState(),
	})
	if err != nil {
		return nil, toStatus(err, "oauth callback")
	}

	linkedPages := make([]*impb.ProviderMetaLinkedPage, len(pages))
	for i, p := range pages {
		linkedPages[i] = &impb.ProviderMetaLinkedPage{
			PageId:      p.PageID,
			PageName:    p.PageName,
			AccessToken: p.PageToken,
			Platform:    "facebook",
		}
	}

	return &impb.ProviderMetaOAuthCallbackResponse{
		UserAccessToken: userToken,
		Pages:           linkedPages,
	}, nil
}
