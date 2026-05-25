package service

import (
	"context"
	"log/slog"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
)

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

func (s *MetaAppService) CreateMetaApp(ctx context.Context, req fbmodel.CreateMetaApp) (*fbmodel.MetaApp, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

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

func (s *MetaAppService) GetMetaApp(ctx context.Context, id string) (*fbmodel.MetaApp, error) {
	return s.repo.Select(ctx, id)
}

func (s *MetaAppService) UpdateMetaApp(ctx context.Context, req fbmodel.UpdateMetaApp) (*fbmodel.MetaApp, error) {
	app, err := s.repo.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	req.ApplyTo(app)

	if err := s.repo.Update(ctx, app); err != nil {
		s.log.Error("failed to update meta app", "id", req.ID, "err", err)
		return nil, err
	}

	s.log.Info("meta app updated", "id", app.ID)
	return app, nil
}

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

