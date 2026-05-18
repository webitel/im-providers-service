package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

// -- mock store --

type mockMetaAppStore struct {
	selectFn       func(ctx context.Context, id string) (*model.MetaApp, error)
	selectByURIFn  func(ctx context.Context, uri string) (*model.MetaApp, error)
	insertFn       func(ctx context.Context, a *model.MetaApp) error
	updateFn       func(ctx context.Context, a *model.MetaApp) error
	deleteFn       func(ctx context.Context, id string) error
}

func (m *mockMetaAppStore) Select(ctx context.Context, id string) (*model.MetaApp, error) {
	return m.selectFn(ctx, id)
}
func (m *mockMetaAppStore) SelectByURI(ctx context.Context, uri string) (*model.MetaApp, error) {
	return m.selectByURIFn(ctx, uri)
}
func (m *mockMetaAppStore) Insert(ctx context.Context, a *model.MetaApp) error {
	return m.insertFn(ctx, a)
}
func (m *mockMetaAppStore) Update(ctx context.Context, a *model.MetaApp) error {
	return m.updateFn(ctx, a)
}
func (m *mockMetaAppStore) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

func stubMetaApp() *model.MetaApp {
	return &model.MetaApp{
		ID:               "app-1",
		AppID:            "fb-app-id",
		AppSecret:        "fb-secret",
		OAuthRedirectURI: "https://example.com/callback",
		Scopes:           []string{"pages_messaging", "pages_read_engagement"},
	}
}

// newTestOAuthService builds a MetaOAuthService whose HTTP client points to srv.
func newTestOAuthService(repo store.MetaAppStore, srv *httptest.Server) *MetaOAuthService {
	svc := NewMetaOAuthService(repo, noopLogger)
	svc.client = srv.Client()
	return svc
}

// -- StartOAuth tests --

func TestStartOAuth_Success(t *testing.T) {
	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, id string) (*model.MetaApp, error) {
			if id != "app-1" {
				return nil, store.ErrNotFound
			}
			return stubMetaApp(), nil
		},
	}
	// StartOAuth doesn't make HTTP calls — any server works as placeholder.
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	svc := newTestOAuthService(repo, ts)
	authURL, state, err := svc.StartOAuth(context.Background(), model.OAuthStart{MetaAppID: "app-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(authURL, "facebook.com") {
		t.Errorf("expected facebook.com in auth url, got: %s", authURL)
	}
	if !strings.Contains(authURL, "fb-app-id") {
		t.Errorf("expected client_id in auth url, got: %s", authURL)
	}
	if len(state) != 32 { // 16 random bytes → 32 hex chars
		t.Errorf("unexpected state length: %d", len(state))
	}
}

func TestStartOAuth_AppNotFound(t *testing.T) {
	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, _ string) (*model.MetaApp, error) {
			return nil, store.ErrNotFound
		},
	}
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	svc := newTestOAuthService(repo, ts)
	_, _, err := svc.StartOAuth(context.Background(), model.OAuthStart{MetaAppID: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound wrapped, got: %v", err)
	}
}

func TestStartOAuth_URLContainsScopes(t *testing.T) {
	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, _ string) (*model.MetaApp, error) {
			return stubMetaApp(), nil
		},
	}
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	svc := newTestOAuthService(repo, ts)
	authURL, _, err := svc.StartOAuth(context.Background(), model.OAuthStart{MetaAppID: "app-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(authURL, "pages_messaging") {
		t.Errorf("expected scope in auth url, got: %s", authURL)
	}
}

// -- HandleCallback tests --

// metaTokenResponse simulates graph.facebook.com/v25.0/oauth/access_token
func metaTokenResponse(token string) []byte {
	b, _ := json.Marshal(map[string]string{"access_token": token})
	return b
}

// metaPagesResponse simulates graph.facebook.com/v25.0/me/accounts
func metaPagesResponse(pages []struct{ ID, Name, AccessToken string }) []byte {
	type page struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		AccessToken string `json:"access_token"`
	}
	var data []page
	for _, p := range pages {
		data = append(data, page{ID: p.ID, Name: p.Name, AccessToken: p.AccessToken})
	}
	b, _ := json.Marshal(map[string]interface{}{"data": data})
	return b
}

// buildCallbackServer returns a test server that answers token exchange + pages endpoints.
// callCount tracks how many POST/GET requests it served.
func buildCallbackServer(shortToken, longToken string, pages []struct{ ID, Name, AccessToken string }) (*httptest.Server, *int) {
	count := 0
	mux := http.NewServeMux()

	// Both token exchange requests hit the same path.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		count++
		path := r.URL.Path

		if strings.Contains(path, "oauth/access_token") {
			body := r.FormValue("grant_type")
			if body == "fb_exchange_token" {
				w.Write(metaTokenResponse(longToken))
			} else {
				w.Write(metaTokenResponse(shortToken))
			}
			return
		}

		if strings.Contains(path, "me/accounts") {
			w.Write(metaPagesResponse(pages))
			return
		}

		http.NotFound(w, r)
	})

	srv := httptest.NewServer(mux)
	return srv, &count
}

func TestHandleCallback_Success(t *testing.T) {
	pages := []struct{ ID, Name, AccessToken string }{
		{"page-1", "Page One", "page-tok-1"},
		{"page-2", "Page Two", "page-tok-2"},
	}
	ts, _ := buildCallbackServer("short-tok", "long-tok", pages)
	defer ts.Close()

	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, _ string) (*model.MetaApp, error) {
			app := stubMetaApp()
			// redirect token endpoint to test server
			app.OAuthRedirectURI = ts.URL + "/callback"
			return app, nil
		},
	}

	svc := NewMetaOAuthService(repo, noopLogger)
	// Override the Graph API base to the test server by wrapping the client transport.
	// Since doPOSTTokenRequest and fetchUserPages build absolute URLs from graph.facebook.com,
	// we patch those calls via a transport that rewrites the host.
	svc.client = &http.Client{
		Transport: rewriteHostTransport(ts.URL),
	}

	longTok, fetchedPages, err := svc.HandleCallback(context.Background(), model.OAuthCallback{
		MetaAppID: "app-1",
		Code:      "auth-code",
		State:     "state",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if longTok != "long-tok" {
		t.Errorf("unexpected long token: %s", longTok)
	}
	if len(fetchedPages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(fetchedPages))
	}
	if fetchedPages[0].PageID != "page-1" {
		t.Errorf("unexpected page id: %s", fetchedPages[0].PageID)
	}
}

func TestHandleCallback_AppNotFound(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, _ string) (*model.MetaApp, error) {
			return nil, store.ErrNotFound
		},
	}
	svc := newTestOAuthService(repo, ts)
	_, _, err := svc.HandleCallback(context.Background(), model.OAuthCallback{MetaAppID: "missing"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleCallback_TokenExchangeError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid OAuth access token",
				"type":    "OAuthException",
				"code":    190,
			},
		})
	}))
	defer ts.Close()

	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, _ string) (*model.MetaApp, error) {
			return stubMetaApp(), nil
		},
	}
	svc := NewMetaOAuthService(repo, noopLogger)
	svc.client = &http.Client{Transport: rewriteHostTransport(ts.URL)}

	_, _, err := svc.HandleCallback(context.Background(), model.OAuthCallback{
		MetaAppID: "app-1",
		Code:      "bad-code",
	})
	if err == nil {
		t.Fatal("expected error from meta api, got nil")
	}
}

func TestHandleCallback_EmptyPages(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "me/accounts") {
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"access_token": "tok"})
	}))
	defer ts.Close()

	repo := &mockMetaAppStore{
		selectFn: func(_ context.Context, _ string) (*model.MetaApp, error) {
			return stubMetaApp(), nil
		},
	}
	svc := NewMetaOAuthService(repo, noopLogger)
	svc.client = &http.Client{Transport: rewriteHostTransport(ts.URL)}

	longTok, fetchedPages, err := svc.HandleCallback(context.Background(), model.OAuthCallback{
		MetaAppID: "app-1",
		Code:      "code",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if longTok != "tok" {
		t.Errorf("unexpected token: %s", longTok)
	}
	if len(fetchedPages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(fetchedPages))
	}
}

// -- generateSecureState tests --

func TestGenerateSecureState_Length(t *testing.T) {
	state, err := generateSecureState(16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state) != 32 { // 16 bytes → 32 hex chars
		t.Errorf("expected 32 chars, got %d", len(state))
	}
}

func TestGenerateSecureState_Unique(t *testing.T) {
	a, _ := generateSecureState(16)
	b, _ := generateSecureState(16)
	if a == b {
		t.Error("expected unique states, got identical values")
	}
}

// rewriteHostTransport redirects all outbound requests to baseURL, preserving the path+query.
type rewriteHostTransport string

func (base rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	rewritten := fmt.Sprintf("%s%s", string(base), req.URL.RequestURI())
	parsed, err := clone.URL.Parse(rewritten)
	if err != nil {
		return nil, err
	}
	clone.URL = parsed
	clone.Host = parsed.Host
	return http.DefaultTransport.RoundTrip(clone)
}
