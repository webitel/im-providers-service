package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
)

// [INTERFACE GUARD]
// Ensure gateStore properly implements the store.GateStore interface.
var _ store.GateStore = (*gateStore)(nil)

type gateStore struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// NewGateStore creates a new instance of the gate store with injected dependencies.
func NewGateStore(pool *pgxpool.Pool, cfg *config.Config) store.GateStore {
	return &gateStore{
		pool: pool,
		cfg:  cfg,
	}
}

// List retrieves a paginated list of gate summaries from the database view.
func (s *gateStore) List(ctx context.Context, f model.ListFilter) ([]*model.GateSummary, bool, error) {
	// 1. Setup pagination parameters
	limit := f.Size
	if limit <= 0 {
		limit = 20
	}

	// 2. Execute query against the simplified view (without hardcoded URLs)
	// We fetch limit+1 to check if there is a next page available.
	const query = `SELECT * FROM im_provider.gate_summary ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	var list []*model.GateSummary
	if err := pgxscan.Select(ctx, s.pool, &list, query, limit+1, f.Page*limit); err != nil {
		return nil, false, fmt.Errorf("failed to select gate summaries: %w", err)
	}

	// 3. Populate dynamic fields (WebhookURL) from the application config
	// This keeps the DB domain-agnostic and allows easy URL updates via ENV.
	publicURL := strings.TrimSuffix(s.cfg.Service.PublicURL, "/")
	basePath := strings.Trim(s.cfg.Service.WebhookPath, "/")
	if basePath == "" {
		basePath = "wh"
	}

	for _, item := range list {
		// Use the Stringer-generated String() method of GateType (e.g., "facebook", "whatsapp")
		item.WebhookURL = fmt.Sprintf("%s/im/%s/%s", publicURL, basePath, item.Type)
	}

	// 4. Handle pagination result
	next := len(list) > limit
	if next {
		list = list[:limit]
	}

	return list, next, nil
}

// Delete removes a specific gate. Cascading constraints in DB should handle related configs.
func (s *gateStore) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM im_provider.gates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to execute delete query: %w", err)
	}

	if res.RowsAffected() == 0 {
		return store.ErrNotFound
	}

	return nil
}
