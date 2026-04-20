package service

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

var _ GateManager = (*GateService)(nil)

type GateManager interface {
	ListGates(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error)
}

type GateService struct {
	repo store.GateStore
	log  *slog.Logger
}

func NewGateService(repo store.GateStore, log *slog.Logger) *GateService {
	return &GateService{
		repo: repo,
		log:  log.With("layer", "service", "domain", "gate"),
	}
}

// ListGates coordinates gate retrieval between the store and transport layers.
func (g *GateService) ListGates(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
	return g.repo.List(ctx, f)
}
