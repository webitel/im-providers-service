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
