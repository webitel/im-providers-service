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

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

// [INTERFACE GUARD]
var _ MetaOAuthManager = (*MetaOAuthService)(nil)

type MetaOAuthManager interface {
	StartOAuth(ctx context.Context, req model.OAuthStart) (authURL string, state string, err error)
	HandleCallback(ctx context.Context, req model.OAuthCallback) (longUserToken string, pages []*model.FacebookGate, err error)
}

const metaAPIVersion = "v25.0"

type MetaOAuthService struct {
	repo   store.MetaAppStore
	client *http.Client
	logger *slog.Logger
}

func NewMetaOAuthService(repo store.MetaAppStore, logger *slog.Logger) *MetaOAuthService {
	return &MetaOAuthService{
		repo:   repo,
		logger: logger,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// StartOAuth initiates the flow and logs the generated redirect_uri.
func (s *MetaOAuthService) StartOAuth(ctx context.Context, req model.OAuthStart) (string, string, error) {
	app, err := s.repo.Select(ctx, req.MetaAppID)
	if err != nil {
		return "", "", fmt.Errorf("oauth: app not found: %w", err)
	}

	state, err := generateSecureState(16)
	if err != nil {
		return "", "", fmt.Errorf("oauth: state generation failed: %w", err)
	}

	u, _ := url.Parse(fmt.Sprintf("https://www.facebook.com/%s/dialog/oauth", metaAPIVersion))
	q := u.Query()
	q.Set("client_id", app.AppID)
	q.Set("redirect_uri", app.OAuthRedirectURI)
	q.Set("state", state)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(app.Scopes, ","))
	u.RawQuery = q.Encode()

	s.logger.Debug("OAuth flow started",
		slog.String("app_id", app.AppID),
		slog.String("redirect_uri", app.OAuthRedirectURI),
		slog.String("state", state),
	)

	return u.String(), state, nil
}

// HandleCallback executes the token exchange with extra logging for debugging redirect_uri mismatches.
func (s *MetaOAuthService) HandleCallback(ctx context.Context, req model.OAuthCallback) (string, []*model.FacebookGate, error) {
	app, err := s.repo.Select(ctx, req.MetaAppID)
	if err != nil {
		return "", nil, fmt.Errorf("oauth: config lookup failed: %w", err)
	}

	s.logger.Debug("Attempting code exchange",
		slog.String("meta_app_id", req.MetaAppID),
		slog.String("redirect_uri_from_db", app.OAuthRedirectURI),
		slog.Int("code_length", len(req.Code)),
	)

	// 1. Exchange 'code' using POST
	shortToken, err := s.exchangeCodeForToken(app, req.Code)
	if err != nil {
		s.logger.Error("code exchange failed", slog.Any("err", err))
		return "", nil, err
	}

	// 2. Upgrade to long-lived token
	longUserToken, err := s.upgradeToLongLivedToken(app, shortToken)
	if err != nil {
		s.logger.Error("token upgrade failed", slog.Any("err", err))
		return "", nil, err
	}

	// 3. Fetch pages
	pages, err := s.fetchUserPages(longUserToken)
	if err != nil {
		s.logger.Error("fetching pages failed", slog.Any("err", err))
		return "", nil, err
	}

	s.logger.Info("OAuth callback processed successfully",
		slog.Int("pages_found", len(pages)),
	)

	return longUserToken, pages, nil
}

func (s *MetaOAuthService) exchangeCodeForToken(app *model.MetaApp, code string) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/%s/oauth/access_token", metaAPIVersion)

	val := url.Values{}
	val.Set("client_id", app.AppID)
	val.Set("client_secret", app.AppSecret)
	val.Set("redirect_uri", app.OAuthRedirectURI) // MUST be exactly the same as in StartOAuth
	val.Set("code", code)

	s.logger.Debug("Requesting access token from Meta",
		slog.String("url", apiURL),
		slog.String("sent_redirect_uri", app.OAuthRedirectURI),
	)

	return s.doPOSTTokenRequest(apiURL, val)
}

func (s *MetaOAuthService) upgradeToLongLivedToken(app *model.MetaApp, shortToken string) (string, error) {
	apiURL := fmt.Sprintf("https://graph.facebook.com/%s/oauth/access_token", metaAPIVersion)

	val := url.Values{}
	val.Set("grant_type", "fb_exchange_token")
	val.Set("client_id", app.AppID)
	val.Set("client_secret", app.AppSecret)
	val.Set("fb_exchange_token", shortToken)

	return s.doPOSTTokenRequest(apiURL, val)
}

func (s *MetaOAuthService) doPOSTTokenRequest(apiURL string, val url.Values) (string, error) {
	resp, err := s.client.PostForm(apiURL, val)
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

func (s *MetaOAuthService) fetchUserPages(userToken string) ([]*model.FacebookGate, error) {
	u, _ := url.Parse(fmt.Sprintf("https://graph.facebook.com/%s/me/accounts", metaAPIVersion))
	q := u.Query()
	q.Set("access_token", userToken)
	q.Set("fields", "id,name,access_token")
	u.RawQuery = q.Encode()

	resp, err := s.client.Get(u.String())
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

	var gates []*model.FacebookGate
	for _, p := range result.Data {
		gates = append(gates, &model.FacebookGate{
			PageID:    p.ID,
			Name:      p.Name,
			PageToken: p.AccessToken,
		})
	}
	return gates, nil
}

func generateSecureState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
