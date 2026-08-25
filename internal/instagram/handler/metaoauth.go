package handler

import (
	"context"
	"log/slog"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	igservice "github.com/webitel/im-providers-service/internal/instagram/service"
)

var _ impb.MetaOAuthServiceServer = (*InstagramMetaOAuthHandler)(nil)

type InstagramMetaOAuthHandler struct {
	impb.UnimplementedMetaOAuthServiceServer

	logger *slog.Logger
	srv    igservice.InstagramOAuthManager
}

func NewInstagramMetaOAuthHandler(logger *slog.Logger, srv igservice.InstagramOAuthManager) *InstagramMetaOAuthHandler {
	return &InstagramMetaOAuthHandler{logger: logger, srv: srv}
}

func (h *InstagramMetaOAuthHandler) StartMetaOAuth(ctx context.Context, req *impb.ProviderMetaOAuthStartRequest) (*impb.ProviderMetaOAuthStartResponse, error) {
	authURL, state, err := h.srv.StartOAuth(ctx, igmodel.OAuthStart{
		MetaAppID: req.GetMetaAppId(),
	})
	if err != nil {
		return nil, toStatus(err, "start instagram oauth")
	}

	return &impb.ProviderMetaOAuthStartResponse{
		AuthUrl: authURL,
		State:   state,
	}, nil
}

func (h *InstagramMetaOAuthHandler) MetaOAuthCallback(ctx context.Context, req *impb.ProviderMetaOAuthCallbackRequest) (*impb.ProviderMetaOAuthCallbackResponse, error) {
	userToken, businessAccountID, username, err := h.srv.HandleCallback(ctx, igmodel.OAuthCallback{
		MetaAppID: req.GetMetaAppId(),
		Code:      req.GetCode(),
		State:     req.GetState(),
	})
	if err != nil {
		return nil, toStatus(err, "instagram oauth callback")
	}

	// Instagram Business Login returns a single business account, not a list of pages.
	// Map to the same proto structure for consistency with the Meta OAuth interface.
	linkedPages := []*impb.ProviderMetaLinkedPage{
		{
			PageId:      businessAccountID,
			PageName:    username,
			AccessToken: userToken,
			Platform:    "instagram",
		},
	}

	return &impb.ProviderMetaOAuthCallbackResponse{
		UserAccessToken: userToken,
		Pages:           linkedPages,
	}, nil
}
