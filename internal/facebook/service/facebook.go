package service

import (
	"context"
	"log/slog"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
)

var _ FacebookManager = (*FacebookService)(nil)

type FacebookManager interface {
	CreateGate(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error)
	GetGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
	UpdateGate(ctx context.Context, req fbmodel.UpdateFacebook) (*fbmodel.FacebookGate, error)
	DeleteGate(ctx context.Context, id string) (*fbmodel.FacebookGate, error)
}

type FacebookService struct {
	repo fbstore.FacebookStore
	log  *slog.Logger
}

func NewFacebookService(repo fbstore.FacebookStore, log *slog.Logger) *FacebookService {
	return &FacebookService{
		repo: repo,
		log:  log.With("layer", "service", "domain", "facebook_gate"),
	}
}

func (f *FacebookService) CreateGate(ctx context.Context, req fbmodel.CreateFacebook) (*fbmodel.FacebookGate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	gate := &fbmodel.FacebookGate{
		Name:      req.Name,
		MetaAppID: req.MetaAppID,
		PageID:    req.PageID,
		PageToken: req.PageToken,
		Peer:      req.Peer,
		Enabled:   true,
	}

	if err := f.repo.Insert(ctx, req.Dc, gate); err != nil {
		f.log.Error("failed to create facebook gate", "page_id", req.PageID, semconv.ErrorKey, err)
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

	req.ApplyTo(gate)

	if err := f.repo.Update(ctx, gate); err != nil {
		f.log.Error("failed to update facebook gate", "id", req.ID, semconv.ErrorKey, err)
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
		f.log.Error("failed to unbind facebook gate", "id", id, semconv.ErrorKey, err)
		return nil, err
	}

	f.log.Warn("facebook gate configuration removed", "id", id, "page_id", gate.PageID)
	return gate, nil
}
