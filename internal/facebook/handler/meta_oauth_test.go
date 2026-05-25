package handler

import (
	"context"
	"errors"
	"testing"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
)

type mockMetaOAuthService struct {
	startFn    func(ctx context.Context, req fbmodel.OAuthStart) (string, string, error)
	callbackFn func(ctx context.Context, req fbmodel.OAuthCallback) (string, []fbmodel.LinkedPage, error)
}

func (m *mockMetaOAuthService) StartOAuth(ctx context.Context, req fbmodel.OAuthStart) (string, string, error) {
	return m.startFn(ctx, req)
}

func (m *mockMetaOAuthService) HandleCallback(ctx context.Context, req fbmodel.OAuthCallback) (string, []fbmodel.LinkedPage, error) {
	return m.callbackFn(ctx, req)
}

func TestStartMetaOAuth_Success(t *testing.T) {
	svc := &mockMetaOAuthService{
		startFn: func(_ context.Context, req fbmodel.OAuthStart) (string, string, error) {
			if req.MetaAppID != "app-1" {
				t.Errorf("unexpected meta app id: %s", req.MetaAppID)
			}
			return "https://fb.com/dialog/oauth?...", "secure-state", nil
		},
	}
	h := NewMetaOauthHandler(noopLogger, svc)
	resp, err := h.StartMetaOAuth(context.Background(), &impb.ProviderMetaOAuthStartRequest{
		MetaAppId: "app-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AuthUrl == "" {
		t.Error("expected non-empty auth url")
	}
	if resp.State != "secure-state" {
		t.Errorf("unexpected state: %s", resp.State)
	}
}

func TestStartMetaOAuth_ServiceError(t *testing.T) {
	svc := &mockMetaOAuthService{
		startFn: func(_ context.Context, _ fbmodel.OAuthStart) (string, string, error) {
			return "", "", errors.New("app not found")
		},
	}
	h := NewMetaOauthHandler(noopLogger, svc)
	_, err := h.StartMetaOAuth(context.Background(), &impb.ProviderMetaOAuthStartRequest{MetaAppId: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMetaOAuthCallback_Success(t *testing.T) {
	svc := &mockMetaOAuthService{
		callbackFn: func(_ context.Context, req fbmodel.OAuthCallback) (string, []fbmodel.LinkedPage, error) {
			if req.Code != "auth-code" {
				t.Errorf("unexpected code: %s", req.Code)
			}
			return "long-user-token", []fbmodel.LinkedPage{
				{PageID: "page-1", PageName: "Page One", PageToken: "page-tok"},
				{PageID: "page-2", PageName: "Page Two", PageToken: "page-tok-2"},
			}, nil
		},
	}
	h := NewMetaOauthHandler(noopLogger, svc)
	resp, err := h.MetaOAuthCallback(context.Background(), &impb.ProviderMetaOAuthCallbackRequest{
		MetaAppId: "app-1",
		Code:      "auth-code",
		State:     "state-abc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.UserAccessToken != "long-user-token" {
		t.Errorf("unexpected user access token: %s", resp.UserAccessToken)
	}
	if len(resp.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(resp.Pages))
	}
	if resp.Pages[0].PageId != "page-1" {
		t.Errorf("unexpected page id: %s", resp.Pages[0].PageId)
	}
	if resp.Pages[0].Platform != "facebook" {
		t.Errorf("unexpected platform: %s", resp.Pages[0].Platform)
	}
}

func TestMetaOAuthCallback_EmptyPages(t *testing.T) {
	svc := &mockMetaOAuthService{
		callbackFn: func(_ context.Context, _ fbmodel.OAuthCallback) (string, []fbmodel.LinkedPage, error) {
			return "token", nil, nil
		},
	}
	h := NewMetaOauthHandler(noopLogger, svc)
	resp, err := h.MetaOAuthCallback(context.Background(), &impb.ProviderMetaOAuthCallbackRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(resp.Pages))
	}
}

func TestMetaOAuthCallback_ServiceError(t *testing.T) {
	svc := &mockMetaOAuthService{
		callbackFn: func(_ context.Context, _ fbmodel.OAuthCallback) (string, []fbmodel.LinkedPage, error) {
			return "", nil, errors.New("invalid code")
		},
	}
	h := NewMetaOauthHandler(noopLogger, svc)
	_, err := h.MetaOAuthCallback(context.Background(), &impb.ProviderMetaOAuthCallbackRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
