package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var _ InstagramOAuthManager = (*InstagramOAuthService)(nil)

// InstagramOAuthManager handles Instagram Business Login OAuth flow.
// Reference: https://developers.instagram.com/docs/instagram-business-account-api/authentication
type InstagramOAuthManager interface {
	StartOAuth(ctx context.Context, req igmodel.OAuthStart) (authURL, state string, err error)
	HandleCallback(ctx context.Context, req igmodel.OAuthCallback) (longUserToken, businessAccountID, username string, err error)
}

const (
	// Instagram Business Login endpoints
	// Reference: https://developers.instagram.com/docs/instagram-business-account-api/authentication
	igAuthURL       = "https://api.instagram.com/oauth/authorize"
	igTokenURL      = "https://api.instagram.com/oauth/access_token"
	igGraphURL      = "https://graph.instagram.com"
	igExchangeURL   = igGraphURL + "/access_token"
	igMeURL         = igGraphURL + "/me"
	igDefaultScopes = "instagram_business_basic,instagram_business_manage_messages"
)

type InstagramOAuthService struct {
	repo      fbstore.MetaAppStore
	client    *http.Client
	logger    *slog.Logger
	encryptor crypto.Encryptor
	rdb       *redis.Client
}

func NewInstagramOAuthService(
	repo fbstore.MetaAppStore,
	logger *slog.Logger,
	encryptor crypto.Encryptor,
	rdb *redis.Client,
) *InstagramOAuthService {
	return &InstagramOAuthService{
		repo:      repo,
		logger:    logger,
		encryptor: encryptor,
		rdb:       rdb,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

const (
	oauthStateTTL       = 10 * time.Minute
	oauthStateKeyPrefix = "instagram:oauth:state:"
)

func oauthStateKey(state string) string { return oauthStateKeyPrefix + state }

func (s *InstagramOAuthService) StartOAuth(ctx context.Context, req igmodel.OAuthStart) (string, string, error) {
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
	q.Set("scope", igDefaultScopes)

	u, err := url.Parse(igAuthURL)
	if err != nil {
		return "", "", fmt.Errorf("oauth: parse authorize url: %w", err)
	}

	u.RawQuery = q.Encode()

	// Persist the CSRF state so HandleCallback can prove the code came from a flow
	// we initiated. Fail closed if the store is unavailable rather than issuing an
	// unverifiable state. (rdb is nil only in unit tests.)
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, oauthStateKey(state), "1", oauthStateTTL).Err(); err != nil {
			return "", "", fmt.Errorf("oauth: persist state: %w", err)
		}
	}

	s.logger.Debug("Instagram OAuth flow started",
		slog.String("app_id", app.AppID),
		slog.String("redirect_uri", app.OAuthRedirectURI),
	)

	return u.String(), state, nil
}

func (s *InstagramOAuthService) HandleCallback(ctx context.Context, req igmodel.OAuthCallback) (string, string, string, error) {
	// Verify and single-use consume the CSRF state before doing anything with the
	// code — Del returns the number of keys removed, so an unknown/expired/replayed
	// state yields 0 and is rejected. (rdb is nil only in unit tests.)
	if s.rdb != nil {
		deleted, err := s.rdb.Del(ctx, oauthStateKey(req.State)).Result()
		if err != nil {
			return "", "", "", fmt.Errorf("oauth: state verification failed: %w", err)
		}

		if req.State == "" || deleted == 0 {
			return "", "", "", errors.New("oauth: invalid or expired state")
		}
	}

	app, err := s.repo.Select(ctx, req.MetaAppID)
	if err != nil {
		return "", "", "", fmt.Errorf("oauth: config lookup failed: %w", err)
	}

	s.logger.Debug("Attempting Instagram code exchange",
		slog.String("meta_app_id", req.MetaAppID),
		slog.Int("code_length", len(req.Code)),
	)

	shortToken, err := s.exchangeCodeForToken(ctx, app, req.Code)
	if err != nil {
		return "", "", "", err
	}

	longToken, err := s.upgradeToLongLivedToken(ctx, app, shortToken)
	if err != nil {
		return "", "", "", err
	}

	businessAccountID, username, err := s.fetchBusinessAccount(ctx, longToken)
	if err != nil {
		return "", "", "", err
	}

	s.logger.Info("Instagram OAuth callback processed successfully",
		slog.String("business_account_id", businessAccountID),
		slog.String("username", username),
	)

	return longToken, businessAccountID, username, nil
}

func (s *InstagramOAuthService) exchangeCodeForToken(ctx context.Context, app *fbmodel.MetaApp, code string) (string, error) {
	val := url.Values{}
	val.Set("client_id", app.AppID)
	val.Set("client_secret", app.AppSecret)
	val.Set("grant_type", "authorization_code")
	val.Set("redirect_uri", app.OAuthRedirectURI)
	val.Set("code", code)

	return s.doPOSTTokenRequest(ctx, igTokenURL, val)
}

func (s *InstagramOAuthService) upgradeToLongLivedToken(ctx context.Context, app *fbmodel.MetaApp, shortToken string) (string, error) {
	val := url.Values{}
	val.Set("grant_type", "ig_exchange_token")
	val.Set("client_secret", app.AppSecret)
	val.Set("access_token", shortToken)

	return s.doPOSTTokenRequest(ctx, igExchangeURL, val)
}

func (s *InstagramOAuthService) doPOSTTokenRequest(ctx context.Context, apiURL string, val url.Values) (string, error) {
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

	// Decode best-effort; a non-200 body may carry a structured error or none at
	// all, so the HTTP status is the authoritative failure signal below.
	_ = json.NewDecoder(resp.Body).Decode(&data)

	if data.Error != nil {
		s.logger.Warn("Instagram API error details",
			slog.String("msg", data.Error.Message),
			slog.String("type", data.Error.Type),
			slog.Int("code", data.Error.Code),
		)

		return "", fmt.Errorf("instagram api error: %s", data.Error.Message)
	}

	// A 4xx/5xx without an `error` key must not silently yield an empty token.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("instagram token endpoint %s: unexpected status %s", apiURL, resp.Status)
	}

	if data.AccessToken == "" {
		return "", fmt.Errorf("instagram token endpoint %s: empty access_token", apiURL)
	}

	return data.AccessToken, nil
}

func (s *InstagramOAuthService) fetchBusinessAccount(ctx context.Context, userToken string) (string, string, error) {
	q := url.Values{}
	q.Set("fields", "user_id,username")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, igMeURL+"?"+q.Encode(), nil)
	if err != nil {
		return "", "", err
	}

	// Pass the token as a Bearer header, not a query param, so it never lands in
	// proxy/CDN/access logs.
	req.Header.Set("Authorization", "Bearer "+userToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Error    *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	if result.Error != nil {
		return "", "", fmt.Errorf("instagram api error: %s", result.Error.Message)
	}

	return result.UserID, result.Username, nil
}

func generateSecureState(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
