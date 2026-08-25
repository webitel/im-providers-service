package service

import (
	"context"
	"log/slog"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	igstore "github.com/webitel/im-providers-service/internal/instagram/store"
)

var _ InstagramManager = (*InstagramService)(nil)

type InstagramManager interface {
	CreateGate(ctx context.Context, req igmodel.CreateInstagram) (*igmodel.InstagramGate, error)
	// GetGate and DeleteGate take the caller's domain (dc) so a gate can only be
	// read or removed by the tenant that owns it. UpdateGate scopes via req.Dc.
	GetGate(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error)
	UpdateGate(ctx context.Context, req igmodel.UpdateInstagram) (*igmodel.InstagramGate, error)
	DeleteGate(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error)
}

// ownedBy returns the gate only when it belongs to dc; otherwise ErrNotFound,
// so a caller cannot probe or touch another tenant's gate by UUID.
func ownedBy(gate *igmodel.InstagramGate, dc int64) (*igmodel.InstagramGate, error) {
	if gate.DomainID != dc {
		return nil, sharedstore.ErrNotFound
	}

	return gate, nil
}

type InstagramService struct {
	repo igstore.InstagramStore
	log  *slog.Logger
}

func NewInstagramService(repo igstore.InstagramStore, log *slog.Logger) *InstagramService {
	return &InstagramService{
		repo: repo,
		log:  log.With("layer", "service", "domain", "instagram_gate"),
	}
}

func (i *InstagramService) CreateGate(ctx context.Context, req igmodel.CreateInstagram) (*igmodel.InstagramGate, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	gate := &igmodel.InstagramGate{
		Name:              req.Name,
		MetaAppID:         req.MetaAppID,
		BusinessAccountID: req.BusinessAccountID,
		IGToken:           req.IGToken,
		Peer:              req.Peer,
		Enabled:           true,
	}

	if err := i.repo.Insert(ctx, req.Dc, gate); err != nil {
		i.log.Error("failed to create instagram gate", "business_account_id", req.BusinessAccountID, "err", err)

		return nil, err
	}

	i.log.Info("instagram gate created", "id", gate.ID, "business_account_id", gate.BusinessAccountID)

	return gate, nil
}

func (i *InstagramService) GetGate(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error) {
	gate, err := i.repo.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	return ownedBy(gate, dc)
}

func (i *InstagramService) UpdateGate(ctx context.Context, req igmodel.UpdateInstagram) (*igmodel.InstagramGate, error) {
	gate, err := i.repo.Select(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if _, err := ownedBy(gate, req.Dc); err != nil {
		return nil, err
	}

	req.ApplyTo(gate)

	if err := i.repo.Update(ctx, gate); err != nil {
		i.log.Error("failed to update instagram gate", "id", req.ID, "err", err)

		return nil, err
	}

	i.log.Info("instagram gate updated", "id", gate.ID)

	return gate, nil
}

func (i *InstagramService) DeleteGate(ctx context.Context, id string, dc int64) (*igmodel.InstagramGate, error) {
	gate, err := i.repo.Select(ctx, id)
	if err != nil {
		return nil, err
	}

	if _, err := ownedBy(gate, dc); err != nil {
		return nil, err
	}

	if err := i.repo.Unbind(ctx, id); err != nil {
		i.log.Error("failed to unbind instagram gate", "id", id, "err", err)

		return nil, err
	}

	i.log.Warn("instagram gate configuration removed", "id", id, "business_account_id", gate.BusinessAccountID)

	return gate, nil
}
