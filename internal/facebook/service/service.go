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

	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
)

// --- FacebookManager ---

var _ FacebookManager = (*FacebookService)(nil)

type FacebookManager interface {
	CreateGate(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error)
	GetGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
	UpdateGate(ctx context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error)
	DeleteGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
}

type FacebookService struct {
	repo      fbstore.FacebookStore
	contacter *imcontact.Client
	log       *slog.Logger
}

func NewFacebookService(repo fbstore.FacebookStore, contacter *imcontact.Client, log *slog.Logger) *FacebookService {
	return &FacebookService{
		repo:      repo,
		contacter: contacter,
		log:       log.With("layer", "service", "domain", "facebook_gate"),
	}
}

func (f *FacebookService) CreateGate(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error) {
	gate := &fbmodel.FacebookGate{
		Name:      req.Name,
		MetaAppID: req.MetaAppID,
		PageID:    req.PageID,
		PageToken: req.PageToken,
		Peer:      req.Peer,
		Enabled:   true,
	}

	if err := f.repo.Insert(ctx, req.Dc, gate); err != nil {
		f.log.Error("failed to create facebook gate", "page_id", req.PageID, "err", err)
		return nil, err
	}

	f.log.Info("facebook gate created", "id", gate.ID, "page_name", gate.Name)
	return gate, nil
}

func (f *FacebookService) GetGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error) {
	return f.repo.Select(ctx, id)
}

func (f *FacebookService) UpdateGate(ctx context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error) {
	gate, err := f.repo.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		gate.Name = *req.Name
	}
	if req.Enabled != nil {
		gate.Enabled = *req.Enabled
	}
	if req.PageToken != nil {
		gate.PageToken = *req.PageToken
	}
	if req.Peer != nil {
		gate.Peer = *req.Peer
	}

	if err := f.repo.Update(ctx, gate); err != nil {
		f.log.Error("failed to update facebook gate", "id", req.ID, "err", err)
		return nil, err
	}

	f.log.Info("facebook gate updated", "id", gate.ID)
	return gate, nil
}

func (f *FacebookService) DeleteGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error) {
	gate, err := f.repo.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	// Unbind only removes the Facebook-specific configuration (the "tab")
	if err := f.repo.Unbind(ctx, id); err != nil {
		f.log.Error("failed to unbind facebook gate", "id", id, "err", err)
		return nil, err
	}

	f.log.Warn("facebook gate configuration removed", "id", id, "page_id", gate.PageID)
	return gate, nil
}

// --- MetaAppManager ---

// [INTERFACE GUARD]
var _ MetaAppManager = (*MetaAppService)(nil)

type MetaAppManager interface {
	CreateMetaApp(ctx context.Context, req fbmodel.CreateMetaApp) (*fbmodel.MetaApp, error)
	GetMetaApp(ctx context.Context, id string) (*fbmodel.MetaApp, error)
	UpdateMetaApp(ctx context.Context, req fbmodel.UpdateMetaApp) (*fbmodel.MetaApp, error)
	DeleteMetaApp(ctx context.Context, id string) (*fbmodel.MetaApp, error)
}

type MetaAppService struct {
	repo fbstore.MetaAppStore
	log  *slog.Logger
}

func NewMetaAppService(repo fbstore.MetaAppStore, log *slog.Logger) *MetaAppService {
	return &MetaAppService{
		repo: repo,
		log:  log.With("layer", "service", "domain", "meta_app"),
	}
}

// CreateMetaApp validates and stores a new Meta App configuration.
func (s *MetaAppService) CreateMetaApp(ctx context.Context, req fbmodel.CreateMetaApp) (*fbmodel.MetaApp, error) {
	app := &fbmodel.MetaApp{
		Name:             req.Name,
		URI:              req.URI,
		AppID:            req.AppID,
		AppSecret:        req.AppSecret,
		OAuthRedirectURI: req.OAuthRedirectURI,
		Scopes:           req.Scopes,
		VerifyToken:      req.VerifyToken,
	}

	if err := s.repo.Insert(ctx, app); err != nil {
		s.log.Error("failed to create meta app", "app_id", req.AppID, "err", err)
		return nil, err
	}

	s.log.Info("meta app created", "id", app.ID, "app_id", app.AppID)
	return app, nil
}

// GetMetaApp retrieves a single record by its internal UUID.
func (s *MetaAppService) GetMetaApp(ctx context.Context, id string) (*fbmodel.MetaApp, error) {
	return s.repo.Select(ctx, id)
}

// UpdateMetaApp applies a partial update with a version check (Optimistic Locking).
func (s *MetaAppService) UpdateMetaApp(ctx context.Context, req fbmodel.UpdateMetaApp) (*fbmodel.MetaApp, error) {
	// 1. Fetch current state to get existing fields and current UpdatedAt timestamp
	app, err := s.repo.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	// 2. Apply changes from request
	patchMetaModel(app, req)

	// 3. Persist changes. The repo will check if UpdatedAt still matches to prevent race conditions.
	if err := s.repo.Update(ctx, app); err != nil {
		s.log.Error("failed to update meta app", "id", req.ID, "err", err)
		return nil, err
	}

	s.log.Info("meta app updated", "id", app.ID)
	return app, nil
}

// DeleteMetaApp removes the app and returns its final state.
func (s *MetaAppService) DeleteMetaApp(ctx context.Context, id string) (*fbmodel.MetaApp, error) {
	app, err := s.repo.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error("failed to delete meta app", "id", id, "err", err)
		return nil, err
	}

	s.log.Warn("meta app deleted", "id", id)
	return app, nil
}

func patchMetaModel(app *fbmodel.MetaApp, req fbmodel.UpdateMetaApp) {
	if req.Name != nil {
		app.Name = *req.Name
	}
	if req.AppSecret != nil {
		app.AppSecret = *req.AppSecret
	}
	if req.OAuthRedirectURI != nil {
		app.OAuthRedirectURI = *req.OAuthRedirectURI
	}
	if req.Scopes != nil {
		app.Scopes = req.Scopes
	}
	if req.VerifyToken != nil {
		app.VerifyToken = *req.VerifyToken
	}
}

// --- MetaOAuthManager ---

var _ MetaOAuthManager = (*MetaOAuthService)(nil)

type MetaOAuthManager interface {
	StartOAuth(ctx context.Context, req fbmodel.OAuthStart) (authURL string, state string, err error)
	HandleCallback(ctx context.Context, req fbmodel.OAuthCallback) (longUserToken string, pages []*fbmodel.FacebookGate, err error)
}

const metaAPIVersion = "v25.0"

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

// StartOAuth initiates the flow and logs the generated redirect_uri.
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
		slog.String("state", state),
	)

	return u.String(), state, nil
}

// HandleCallback executes the token exchange with extra logging for debugging redirect_uri mismatches.
func (s *MetaOAuthService) HandleCallback(ctx context.Context, req fbmodel.OAuthCallback) (string, []*fbmodel.FacebookGate, error) {
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

func (s *MetaOAuthService) exchangeCodeForToken(app *fbmodel.MetaApp, code string) (string, error) {
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

func (s *MetaOAuthService) upgradeToLongLivedToken(app *fbmodel.MetaApp, shortToken string) (string, error) {
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

func (s *MetaOAuthService) fetchUserPages(userToken string) ([]*fbmodel.FacebookGate, error) {
	q := url.Values{}
	q.Set("access_token", userToken)
	q.Set("fields", "id,name,access_token")
	u := &url.URL{
		Scheme:   "https",
		Host:     "graph.facebook.com",
		Path:     fmt.Sprintf("/%s/me/accounts", metaAPIVersion),
		RawQuery: q.Encode(),
	}

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

	var gates []*fbmodel.FacebookGate
	for _, p := range result.Data {
		gates = append(gates, &fbmodel.FacebookGate{
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
