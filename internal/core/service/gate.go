package service

import (
	"context"
	"log/slog"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
)

var _ GateManager = (*GateService)(nil)

type GateManager interface {
	ListGates(ctx context.Context, f sharedmodel.ListFilter) ([]*sharedmodel.GateSummary, bool, error)
	GetGate(ctx context.Context, domainID int64, id string) (*sharedmodel.GateSummary, error)
}

type GateService struct {
	repo sharedstore.GateStore
	log  *slog.Logger
}

func NewGateService(repo sharedstore.GateStore, log *slog.Logger) *GateService {
	return &GateService{
		repo: repo,
		log:  log.With("layer", "service", "domain", "gate"),
	}
}

// ListGates coordinates gate retrieval between the store and transport layers.
func (g *GateService) ListGates(ctx context.Context, f sharedmodel.ListFilter) ([]*sharedmodel.GateSummary, bool, error) {
	return g.repo.List(ctx, f)
}

// GetGate returns a single gate of the domain.
func (g *GateService) GetGate(ctx context.Context, domainID int64, id string) (*sharedmodel.GateSummary, error) {
	return g.repo.Get(ctx, domainID, id)
}
