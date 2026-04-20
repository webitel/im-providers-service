package service

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

type FacebookManager interface {
	CreateGate(ctx context.Context, req model.CreateFacebook) (*model.FacebookGate, error)
	GetGate(ctx context.Context, id string) (*model.FacebookGate, error)
	UpdateGate(ctx context.Context, req model.UpdateFacebook) (*model.FacebookGate, error)
	DeleteGate(ctx context.Context, id string) (*model.FacebookGate, error)
}

type FacebookService struct {
	repo store.FacebookStore
	log  *slog.Logger
}

func NewFacebookService(repo store.FacebookStore, log *slog.Logger) *FacebookService {
	return &FacebookService{
		repo: repo,
		log:  log.With("layer", "service", "domain", "facebook_gate"),
	}
}

func (f *FacebookService) CreateGate(ctx context.Context, req model.CreateFacebook) (*model.FacebookGate, error) {
	gate := &model.FacebookGate{
		Name:      req.Name,
		MetaAppID: req.MetaAppID,
		PageID:    req.PageID,
		PageToken: req.PageToken,
		Enabled:   true,
	}

	// TODO: Perform Facebook Graph API call to subscribe the app to the page webhooks here.

	if err := f.repo.Insert(ctx, gate); err != nil {
		f.log.Error("failed to create facebook gate", "page_id", req.PageID, "err", err)
		return nil, err
	}

	f.log.Info("facebook gate created", "id", gate.ID, "page_name", gate.Name)
	return gate, nil
}

func (f *FacebookService) GetGate(ctx context.Context, id string) (*model.FacebookGate, error) {
	return f.repo.Select(ctx, id)
}

func (f *FacebookService) UpdateGate(ctx context.Context, req model.UpdateFacebook) (*model.FacebookGate, error) {
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

	if err := f.repo.Update(ctx, gate); err != nil {
		f.log.Error("failed to update facebook gate", "id", req.ID, "err", err)
		return nil, err
	}

	f.log.Info("facebook gate updated", "id", gate.ID) // Use f.log to keep context
	return gate, nil
}

func (f *FacebookService) DeleteGate(ctx context.Context, id string) (*model.FacebookGate, error) {
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
