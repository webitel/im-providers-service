package service

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var noopLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// -- mock store --

type mockMetaAppStore struct {
	selectFn      func(ctx context.Context, id string) (*fbmodel.MetaApp, error)
	selectByURIFn func(ctx context.Context, uri string) (*fbmodel.MetaApp, error)
	insertFn      func(ctx context.Context, a *fbmodel.MetaApp) error
	updateFn      func(ctx context.Context, a *fbmodel.MetaApp) error
	deleteFn      func(ctx context.Context, id string) error
}

func (m *mockMetaAppStore) Select(ctx context.Context, id string) (*fbmodel.MetaApp, error) {
	return m.selectFn(ctx, id)
}

func (m *mockMetaAppStore) SelectByURI(ctx context.Context, uri string) (*fbmodel.MetaApp, error) {
	return m.selectByURIFn(ctx, uri)
}

func (m *mockMetaAppStore) Insert(ctx context.Context, a *fbmodel.MetaApp) error {
	return m.insertFn(ctx, a)
}

func (m *mockMetaAppStore) Update(ctx context.Context, a *fbmodel.MetaApp) error {
	return m.updateFn(ctx, a)
}

func (m *mockMetaAppStore) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

var _ fbstore.MetaAppStore = (*mockMetaAppStore)(nil)

// -- mock encryptor --

type mockEncryptor struct {
	encryptFn func(plaintext string) (string, error)
	decryptFn func(ciphertext string) (string, error)
}

func (m *mockEncryptor) Encrypt(plaintext string) (string, error) {
	return m.encryptFn(plaintext)
}

func (m *mockEncryptor) Decrypt(ciphertext string) (string, error) {
	return m.decryptFn(ciphertext)
}

var _ crypto.Encryptor = (*mockEncryptor)(nil)

func stubInstagramMetaApp() *fbmodel.MetaApp {
	return &fbmodel.MetaApp{
		ID:               "ig-app-1",
		AppID:            "ig-app-id",
		AppSecret:        "ig-secret",
		OAuthRedirectURI: "https://example.com/callback",
	}
}

// -- StartOAuth tests --

func TestInstagramStartOAuth_Success(t *testing.T) {
	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, id string) (*fbmodel.MetaApp, error) {
			if id != "ig-app-1" {
				return nil, sharedstore.ErrNotFound
			}

			return stubInstagramMetaApp(), nil
		},
	}

	encryptor := &mockEncryptor{
		encryptFn: func(plaintext string) (string, error) { return plaintext, nil },
	}

	svc := NewInstagramOAuthService(repo, noopLogger, encryptor, nil)

	authURL, state, err := svc.StartOAuth(context.Background(), igmodel.OAuthStart{MetaAppID: "ig-app-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(authURL, "api.instagram.com") {
		t.Errorf("expected api.instagram.com in auth url, got: %s", authURL)
	}

	if !strings.Contains(authURL, "ig-app-id") {
		t.Errorf("expected client_id in auth url, got: %s", authURL)
	}

	if !strings.Contains(authURL, "instagram_business_basic") {
		t.Errorf("expected scopes in auth url, got: %s", authURL)
	}

	if len(state) != 32 { // 16 random bytes → 32 hex chars
		t.Errorf("unexpected state length: %d", len(state))
	}
}

func TestInstagramStartOAuth_AppNotFound(t *testing.T) {
	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, _ string) (*fbmodel.MetaApp, error) {
			return nil, sharedstore.ErrNotFound
		},
	}

	encryptor := &mockEncryptor{}
	svc := NewInstagramOAuthService(repo, noopLogger, encryptor, nil)

	_, _, err := svc.StartOAuth(context.Background(), igmodel.OAuthStart{MetaAppID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for missing app")
	}

	if !strings.Contains(err.Error(), "app not found") {
		t.Errorf("expected 'app not found' in error, got: %v", err)
	}
}
