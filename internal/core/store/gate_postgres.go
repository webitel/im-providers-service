package store

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/im-providers-service/config"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// [INTERFACE GUARD]
var _ GateStore = (*gateStore)(nil)

type gateStore struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// NewGateStore creates a new instance of the gate store.
func NewGateStore(pool *pgxpool.Pool, cfg *config.Config) GateStore {
	return &gateStore{
		pool: pool,
		cfg:  cfg,
	}
}

// List retrieves a paginated list of gate summaries from the database view.
func (s *gateStore) List(ctx context.Context, f sharedmodel.ListFilter) ([]*sharedmodel.GateSummary, bool, error) {
	limit := f.Size
	if limit <= 0 {
		limit = 20
	}

	// Offset calculation
	offset := f.Page * limit

	// Fetch limit+1 to determine if there's a next page
	const query = `SELECT * FROM im_provider.gate_summary ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	var list []*sharedmodel.GateSummary
	if err := pgxscan.Select(ctx, s.pool, &list, query, limit+1, offset); err != nil {
		return nil, false, fmt.Errorf("postgres: list gates: %w", err)
	}

	// Check for next page
	next := len(list) > limit
	if next {
		list = list[:limit]
	}

	// WebhookURL generation is removed as it's now handled by the gateway layer or dynamically.
	return list, next, nil
}

// Delete removes a specific gate. Cascading constraints in DB handle bots and configs.
func (s *gateStore) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM im_provider.gates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres: delete gate: %w", err)
	}

	if res.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
