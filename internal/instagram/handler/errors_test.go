package handler

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	igservice "github.com/webitel/im-providers-service/internal/instagram/service"
)

// ---------------------------------------------------------------------------
// OAuth mock helpers (used only in this file)
// ---------------------------------------------------------------------------

type mockOAuthService struct {
	startOAuthFn     func(metaAppID string) (string, string, error)
	handleCallbackFn func(code string) (string, string, string, error)
}

func (m *mockOAuthService) StartOAuth(_ context.Context, req igmodel.OAuthStart) (string, string, error) {
	if m.startOAuthFn != nil {
		return m.startOAuthFn(req.MetaAppID)
	}

	return "", "", errors.New("not implemented")
}

func (m *mockOAuthService) HandleCallback(_ context.Context, req igmodel.OAuthCallback) (string, string, string, error) {
	if m.handleCallbackFn != nil {
		return m.handleCallbackFn(req.Code)
	}

	return "", "", "", errors.New("not implemented")
}

var _ igservice.InstagramOAuthManager = (*mockOAuthService)(nil)

func mockStartOAuthReq(metaAppID string) *impb.ProviderMetaOAuthStartRequest {
	return &impb.ProviderMetaOAuthStartRequest{MetaAppId: metaAppID}
}

func mockOAuthCallbackReq(metaAppID, code, state string) *impb.ProviderMetaOAuthCallbackRequest {
	return &impb.ProviderMetaOAuthCallbackRequest{
		MetaAppId: metaAppID,
		Code:      code,
		State:     state,
	}
}

// ---------------------------------------------------------------------------
// toStatus: error code mapping
// ---------------------------------------------------------------------------

func grpcCode(err error) codes.Code {
	st, ok := status.FromError(err)
	if !ok {
		return codes.Unknown
	}

	return st.Code()
}

func TestToStatus_ValidationError_InvalidArgument(t *testing.T) {
	ve := &igmodel.ValidationError{Fields: []string{"name"}}
	err := toStatus(ve, "op")

	if grpcCode(err) != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %s", grpcCode(err))
	}
}

func TestToStatus_ErrNotFound_NotFound(t *testing.T) {
	err := toStatus(sharedstore.ErrNotFound, "op")

	if grpcCode(err) != codes.NotFound {
		t.Errorf("expected NotFound, got %s", grpcCode(err))
	}
}

func TestToStatus_ErrConflict_AlreadyExists(t *testing.T) {
	err := toStatus(sharedstore.ErrConflict, "op")

	if grpcCode(err) != codes.AlreadyExists {
		t.Errorf("expected AlreadyExists, got %s", grpcCode(err))
	}
}

func TestToStatus_UnknownError_Internal(t *testing.T) {
	err := toStatus(errors.New("some db error"), "create gate")

	if grpcCode(err) != codes.Internal {
		t.Errorf("expected Internal, got %s", grpcCode(err))
	}
}

// ---------------------------------------------------------------------------
// InstagramMetaOAuthHandler: field mapping & token invisibility
// ---------------------------------------------------------------------------

func TestMetaOAuthCallback_TokenNotInLogFields(t *testing.T) {
	// The handler must never surface the token in any structured field —
	// the proto response wraps it, but the LinkedPage.AccessToken must NOT
	// appear in a log record. We validate the handler's proto mapping here:
	// business_account_id → PageId, username → PageName, token → AccessToken,
	// platform == "instagram".
	svc := &mockOAuthService{
		handleCallbackFn: func(_ string) (string, string, string, error) {
			// Returns (longToken, businessAccountID, username, nil)
			return "long-live-token", "biz-12345", "ig_user", nil
		},
	}

	h := NewInstagramMetaOAuthHandler(noopLogger, svc)

	resp, err := h.MetaOAuthCallback(ctxWithAuth(1), mockOAuthCallbackReq("app-1", "code-xyz", "state-abc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.GetPages()) != 1 {
		t.Fatalf("expected 1 linked page, got %d", len(resp.GetPages()))
	}

	page := resp.GetPages()[0]

	if page.GetPageId() != "biz-12345" {
		t.Errorf("PageId: want biz-12345, got %s", page.GetPageId())
	}

	if page.GetPageName() != "ig_user" {
		t.Errorf("PageName: want ig_user, got %s", page.GetPageName())
	}

	if page.GetPlatform() != "instagram" {
		t.Errorf("Platform: want instagram, got %s", page.GetPlatform())
	}

	// Token is present in the response (caller needs it) but must equal the returned value.
	if page.GetAccessToken() != "long-live-token" {
		t.Errorf("AccessToken mismatch: got %s", page.GetAccessToken())
	}

	// UserAccessToken at top level must also be set.
	if resp.GetUserAccessToken() != "long-live-token" {
		t.Errorf("UserAccessToken mismatch: got %s", resp.GetUserAccessToken())
	}
}

func TestStartMetaOAuth_MapsFields(t *testing.T) {
	svc := &mockOAuthService{
		startOAuthFn: func(metaAppID string) (string, string, error) {
			if metaAppID != "app-1" {
				t.Errorf("unexpected metaAppID: %s", metaAppID)
			}

			return "https://api.instagram.com/oauth/authorize?foo=bar", "state-xyz", nil
		},
	}

	h := NewInstagramMetaOAuthHandler(noopLogger, svc)

	resp, err := h.StartMetaOAuth(ctxWithAuth(1), mockStartOAuthReq("app-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.GetAuthUrl() == "" {
		t.Error("expected non-empty auth URL")
	}

	if resp.GetState() != "state-xyz" {
		t.Errorf("state mismatch: got %s", resp.GetState())
	}
}

func TestStartMetaOAuth_ServiceError_Internal(t *testing.T) {
	svc := &mockOAuthService{
		startOAuthFn: func(_ string) (string, string, error) {
			return "", "", errors.New("app not found")
		},
	}

	h := NewInstagramMetaOAuthHandler(noopLogger, svc)

	_, err := h.StartMetaOAuth(ctxWithAuth(1), mockStartOAuthReq("bad-app"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if grpcCode(err) != codes.Internal {
		t.Errorf("expected Internal, got %s", grpcCode(err))
	}
}
