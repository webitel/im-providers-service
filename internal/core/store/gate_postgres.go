package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
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

	offset := pageOffset(f.Page, limit)

	query, args := buildListQuery(f, limit+1, offset)

	var list []*sharedmodel.GateSummary
	if err := pgxscan.Select(ctx, s.pool, &list, query, args...); err != nil {
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

// pageOffset converts a 1-based page (as documented on ListGatesRequest) into a
// row offset. Page 0 is treated as the first page for callers that never set it.
func pageOffset(page, limit int) int {
	if page <= 1 {
		return 0
	}

	return (page - 1) * limit
}

// buildListQuery renders the domain-scoped gate listing with the optional
// type / status / name filters. It fetches limit rows (callers pass size+1 to
// detect a next page).
func buildListQuery(f sharedmodel.ListFilter, limit, offset int) (string, []any) {
	var (
		where = []string{"dc = $1"}
		args  = []any{f.DomainID}
	)

	if len(f.Types) > 0 {
		types := make([]string, len(f.Types))
		for i, t := range f.Types {
			types[i] = t.String()
		}

		args = append(args, types)
		where = append(where, fmt.Sprintf("type = ANY($%d)", len(args)))
	}

	if f.Status != sharedmodel.StatusUnknown {
		args = append(args, f.Status.String())
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}

	if f.Q != "" {
		args = append(args, "%"+f.Q+"%")
		where = append(where, fmt.Sprintf("name ILIKE $%d", len(args)))
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(
		`SELECT %s FROM im_provider.gate_summary_ext WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		gateSummaryColumns, strings.Join(where, " AND "), len(args)-1, len(args),
	)

	return query, args
}

// gateSummaryColumns is the explicit column list of gate_summary_ext. Never
// SELECT * into a struct: scany rejects unknown columns, so any new view
// column would break every build that is already deployed.
const gateSummaryColumns = `id, name, type, status, contact, provider_app_id, created_at, updated_at, dc, address`

// Get returns a single gate summary scoped to the domain.
func (s *gateStore) Get(ctx context.Context, domainID int64, id string) (*sharedmodel.GateSummary, error) {
	const query = `SELECT ` + gateSummaryColumns + ` FROM im_provider.gate_summary_ext WHERE id = $1 AND dc = $2`

	var gate sharedmodel.GateSummary
	if err := pgxscan.Get(ctx, s.pool, &gate, query, id, domainID); err != nil {
		if pgxscan.NotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("postgres: get gate: %w", err)
	}

	return &gate, nil
}

// GetTypeByID returns the provider type for a gate by its UUID.
func (s *gateStore) GetTypeByID(ctx context.Context, id string) (sharedmodel.GateType, error) {
	var typeStr string
	err := s.pool.QueryRow(ctx, `SELECT type FROM im_provider.gate_summary WHERE id = $1`, id).Scan(&typeStr)
	if errors.Is(err, pgx.ErrNoRows) {
		return sharedmodel.TypeUnknown, ErrNotFound
	}
	if err != nil {
		return sharedmodel.TypeUnknown, fmt.Errorf("postgres: get gate type: %w", err)
	}
	return sharedmodel.ParseGateType(typeStr), nil
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
