package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ TemplateStore = (*templateStore)(nil)

type templateStore struct {
	pool *pgxpool.Pool
}

func NewTemplateStore(pool *pgxpool.Pool) TemplateStore {
	return &templateStore{pool: pool}
}

// GetTemplate retrieves the gate-specific template override.
// Returns ErrNotFound when no row exists for (gateID, eventType).
func (s *templateStore) GetTemplate(ctx context.Context, gateID, eventType string) (string, error) {
	const q = `
		SELECT template
		  FROM im_provider.sys_msg_templates
		 WHERE gate_id = $1
		   AND event_type = $2
		 LIMIT 1`

	var tpl string
	err := s.pool.QueryRow(ctx, q, gateID, eventType).Scan(&tpl)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return tpl, nil
}

// SetTemplate creates or replaces the template for (gateID, eventType).
func (s *templateStore) SetTemplate(ctx context.Context, gateID, eventType, template string, domainID int64) error {
	const q = `
		INSERT INTO im_provider.sys_msg_templates (gate_id, event_type, template, domain_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (gate_id, event_type)
		DO UPDATE SET template = EXCLUDED.template, updated_at = NOW()`

	_, err := s.pool.Exec(ctx, q, gateID, eventType, template, domainID)
	return err
}

// DeleteTemplate removes the override. Returns ErrNotFound if no row existed.
func (s *templateStore) DeleteTemplate(ctx context.Context, gateID, eventType string) error {
	const q = `
		DELETE FROM im_provider.sys_msg_templates
		 WHERE gate_id = $1
		   AND event_type = $2`

	tag, err := s.pool.Exec(ctx, q, gateID, eventType)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListTemplates returns all overrides for the given gate.
func (s *templateStore) ListTemplates(ctx context.Context, gateID string) ([]TemplateRow, error) {
	const q = `
		SELECT gate_id, event_type, template
		  FROM im_provider.sys_msg_templates
		 WHERE gate_id = $1
		 ORDER BY event_type`

	rows, err := s.pool.Query(ctx, q, gateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TemplateRow
	for rows.Next() {
		var r TemplateRow
		if err := rows.Scan(&r.GateID, &r.EventType, &r.Template); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
