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

var _ impb.MetaOAuthServiceServer = (*MetaOauthHandler)(nil)

type MetaOauthHandler struct {
	logger *slog.Logger
	srv    service.MetaOAuthManager
	impb.UnimplementedMetaOAuthServiceServer
}

func NewMetaOauthHandler(logger *slog.Logger, srv service.MetaOAuthManager) *MetaOauthHandler {
	return &MetaOauthHandler{
		logger: logger,
		srv:    srv,
	}
}

// StartMetaOAuth initiates the OAuth flow by providing a redirect URL.
func (h *MetaOauthHandler) StartMetaOAuth(ctx context.Context, req *impb.ProviderMetaOAuthStartRequest) (*impb.ProviderMetaOAuthStartResponse, error) {
	authURL, state, err := h.srv.StartOAuth(ctx, model.OAuthStart{
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
	userToken, pages, err := h.srv.HandleCallback(ctx, model.OAuthCallback{
		MetaAppID: req.GetMetaAppId(),
		Code:      req.GetCode(),
		State:     req.GetState(),
	})
	if err != nil {
		h.logger.Error("oauth callback exchange failed", slog.Any("err", err))
		return nil, status.Errorf(codes.Unauthenticated, "failed to exchange authorization code: %v", err)
	}

	// Map internal model list to Protobuf messages
	linkedPages := make([]*impb.ProviderMetaLinkedPage, len(pages))
	for i, p := range pages {
		linkedPages[i] = &impb.ProviderMetaLinkedPage{
			PageId:   p.PageID,
			PageName: p.Name,
			Platform: "facebook",
		}
	}

	return &impb.ProviderMetaOAuthCallbackResponse{
		UserAccessToken: userToken,
		Pages:           linkedPages,
	}, nil
}
