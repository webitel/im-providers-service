package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
)

var _ MetaOAuthManager = (*MetaOAuthService)(nil)

type MetaOAuthManager interface {
	StartOAuth(ctx context.Context, req fbmodel.OAuthStart) (authURL string, state string, err error)
	HandleCallback(ctx context.Context, req fbmodel.OAuthCallback) (longUserToken string, pages []fbmodel.LinkedPage, err error)
}

const (
	metaAPIVersion = "v25.0"
	// metaTokenURL is the Graph API endpoint for both short-lived code exchange
	// and long-lived token upgrade; both operations share the same URL.
	metaTokenURL = "https://graph.facebook.com/" + metaAPIVersion + "/oauth/access_token"
)

type MetaOAuthService struct {
	repo   fbstore.MetaAppStore
	client *http.Client
	logger *slog.Logger
}

func NewMetaOAuthService(repo fbstore.MetaAppStore, logger *slog.Logger) *MetaOAuthService {
	return &MetaOAuthService{
		repo:   repo,
		logger: logger,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *MetaOAuthService) StartOAuth(ctx context.Context, req fbmodel.OAuthStart) (string, string, error) {
	app, err := s.repo.Select(ctx, req.MetaAppID)
	if err != nil {
		return "", "", fmt.Errorf("oauth: app not found: %w", err)
	}

	state, err := generateSecureState(16)
	if err != nil {
		return "", "", fmt.Errorf("oauth: state generation failed: %w", err)
	}

	q := url.Values{}
	q.Set("client_id", app.AppID)
	q.Set("redirect_uri", app.OAuthRedirectURI)
	q.Set("state", state)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(app.Scopes, ","))
	u := &url.URL{
		Scheme:   "https",
		Host:     "www.facebook.com",
		Path:     fmt.Sprintf("/%s/dialog/oauth", metaAPIVersion),
		RawQuery: q.Encode(),
	}

	s.logger.Debug("OAuth flow started",
		slog.String("app_id", app.AppID),
		slog.String("redirect_uri", app.OAuthRedirectURI),
	)

	return u.String(), state, nil
}

func (s *MetaOAuthService) HandleCallback(ctx context.Context, req fbmodel.OAuthCallback) (string, []fbmodel.LinkedPage, error) {
	app, err := s.repo.Select(ctx, req.MetaAppID)
	if err != nil {
		return "", nil, fmt.Errorf("oauth: config lookup failed: %w", err)
	}

	s.logger.Debug("Attempting code exchange",
		slog.String("meta_app_id", req.MetaAppID),
		slog.String("redirect_uri_from_db", app.OAuthRedirectURI),
		slog.Int("code_length", len(req.Code)),
	)

	shortToken, err := s.exchangeCodeForToken(ctx, app, req.Code)
	if err != nil {
		return "", nil, err
	}

	longUserToken, err := s.upgradeToLongLivedToken(ctx, app, shortToken)
	if err != nil {
		return "", nil, err
	}

	pages, err := s.fetchUserPages(ctx, longUserToken)
	if err != nil {
		return "", nil, err
	}

	s.logger.Info("OAuth callback processed successfully",
		slog.Int("pages_found", len(pages)),
	)

	return longUserToken, pages, nil
}

func (s *MetaOAuthService) exchangeCodeForToken(ctx context.Context, app *fbmodel.MetaApp, code string) (string, error) {
	val := url.Values{}
	val.Set("client_id", app.AppID)
	val.Set("client_secret", app.AppSecret)
	val.Set("redirect_uri", app.OAuthRedirectURI) // MUST match the URI used in StartOAuth
	val.Set("code", code)

	return s.doPOSTTokenRequest(ctx, metaTokenURL, val)
}

func (s *MetaOAuthService) upgradeToLongLivedToken(ctx context.Context, app *fbmodel.MetaApp, shortToken string) (string, error) {
	val := url.Values{}
	val.Set("grant_type", "fb_exchange_token")
	val.Set("client_id", app.AppID)
	val.Set("client_secret", app.AppSecret)
	val.Set("fb_exchange_token", shortToken)

	return s.doPOSTTokenRequest(ctx, metaTokenURL, val)
}

func (s *MetaOAuthService) doPOSTTokenRequest(ctx context.Context, apiURL string, val url.Values) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(val.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data struct {
		AccessToken string `json:"access_token"`
		Error       *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Error != nil {
		s.logger.Warn("Meta API error details",
			slog.String("msg", data.Error.Message),
			slog.String("type", data.Error.Type),
			slog.Int("code", data.Error.Code),
		)
		return "", fmt.Errorf("meta api error: %s", data.Error.Message)
	}

	return data.AccessToken, nil
}

func (s *MetaOAuthService) fetchUserPages(ctx context.Context, userToken string) ([]fbmodel.LinkedPage, error) {
	q := url.Values{}
	q.Set("access_token", userToken)
	q.Set("fields", "id,name,access_token")
	u := &url.URL{
		Scheme:   "https",
		Host:     "graph.facebook.com",
		Path:     fmt.Sprintf("/%s/me/accounts", metaAPIVersion),
		RawQuery: q.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	pages := make([]fbmodel.LinkedPage, len(result.Data))
	for i, p := range result.Data {
		pages[i] = fbmodel.LinkedPage{
			PageID:    p.ID,
			PageName:  p.Name,
			PageToken: p.AccessToken,
		}
	}
	return pages, nil
}

func generateSecureState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
