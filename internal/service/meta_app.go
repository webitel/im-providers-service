package service

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

// [INTERFACE GUARD]
var _ MetaAppManager = (*MetaAppService)(nil)

type MetaAppManager interface {
	CreateMetaApp(ctx context.Context, req model.CreateMetaApp) (*model.MetaApp, error)
	GetMetaApp(ctx context.Context, id string) (*model.MetaApp, error)
	UpdateMetaApp(ctx context.Context, req model.UpdateMetaApp) (*model.MetaApp, error)
	DeleteMetaApp(ctx context.Context, id string) (*model.MetaApp, error)
}

type MetaAppService struct {
	repo store.MetaAppStore
	log  *slog.Logger
}

func NewMetaAppService(repo store.MetaAppStore, log *slog.Logger) *MetaAppService {
	return &MetaAppService{
		repo: repo,
		log:  log.With("layer", "service", "domain", "meta_app"),
	}
}

// CreateMetaApp validates and stores a new Meta App configuration.
func (s *MetaAppService) CreateMetaApp(ctx context.Context, req model.CreateMetaApp) (*model.MetaApp, error) {
	app := &model.MetaApp{
		Name:             req.Name,
		AppID:            req.AppID,
		AppSecret:        req.AppSecret,
		OAuthRedirectURI: req.OAuthRedirectURI,
		Scopes:           req.Scopes,
	}

	if err := s.repo.Insert(ctx, app); err != nil {
		s.log.Error("failed to create meta app", "app_id", req.AppID, "err", err)
		return nil, err
	}

	s.log.Info("meta app created", "id", app.ID, "app_id", app.AppID)
	return app, nil
}

// GetMetaApp retrieves a single record by its internal UUID.
func (s *MetaAppService) GetMetaApp(ctx context.Context, id string) (*model.MetaApp, error) {
	return s.repo.Select(ctx, id)
}

// UpdateMetaApp applies a partial update with a version check (Optimistic Locking).
func (s *MetaAppService) UpdateMetaApp(ctx context.Context, req model.UpdateMetaApp) (*model.MetaApp, error) {
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
func (s *MetaAppService) DeleteMetaApp(ctx context.Context, id string) (*model.MetaApp, error) {
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

func patchMetaModel(app *model.MetaApp, req model.UpdateMetaApp) {
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
}
