package postgres

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

// [INTERFACE GUARD]
var _ store.FacebookStore = (*facebookStore)(nil)

type facebookStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

func NewFacebookStore(pool *pgxpool.Pool, crypt crypto.Encryptor) store.FacebookStore {
	return &facebookStore{
		pool:   pool,
		crypto: crypt,
	}
}

func (s *facebookStore) Insert(ctx context.Context, g *model.FacebookGate) error {
	token, err := s.crypto.Encrypt(g.PageToken)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	const query = `
	WITH new_gate AS (
		INSERT INTO im_provider.gates (name, type, enabled)
		VALUES ($1, 'facebook', $2)
		RETURNING id, name, created_at, updated_at
	)
	INSERT INTO im_provider.gate_facebook (gate_id, meta_app_id, page_id, page_token)
	SELECT id, $3, $4, $5 FROM new_gate
	RETURNING 
		gate_id, 
		(SELECT name FROM new_gate),
		(SELECT created_at FROM new_gate), 
		(SELECT updated_at FROM new_gate)`

	// Use temporary variables for time.Time scan to ensure compatibility
	err = s.pool.QueryRow(ctx, query,
		g.Name, g.Enabled, g.MetaAppID, g.PageID, token,
	).Scan(&g.ID, &g.Name, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: insert facebook gate: %w", err)
	}

	s.mapVirtualFields(g)
	return nil
}

func (s *facebookStore) Select(ctx context.Context, id string) (*model.FacebookGate, error) {
	const query = `
	SELECT 
		g.id, g.name, g.enabled, g.created_at, g.updated_at, 
		fb.meta_app_id, fb.page_id, fb.page_token
	FROM im_provider.gates g
	JOIN im_provider.gate_facebook fb ON g.id = fb.gate_id
	WHERE g.id = $1`

	var g model.FacebookGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, id); err != nil {
		if pgxscan.NotFound(err) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	if dec, err := s.crypto.Decrypt(g.PageToken); err == nil {
		g.PageToken = dec
	}

	s.mapVirtualFields(&g)
	return &g, nil
}

func (s *facebookStore) Update(ctx context.Context, g *model.FacebookGate) error {
	token, err := s.crypto.Encrypt(g.PageToken)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		const uGate = `UPDATE im_provider.gates SET name = $1, enabled = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`
		if err := tx.QueryRow(ctx, uGate, g.Name, g.Enabled, g.ID).Scan(&g.UpdatedAt); err != nil {
			return err
		}

		const uConfig = `
			UPDATE im_provider.gate_facebook 
			SET meta_app_id = $1, page_id = $2, page_token = $3 
			WHERE gate_id = $4`
		_, err := tx.Exec(ctx, uConfig, g.MetaAppID, g.PageID, token, g.ID)
		return err
	})
}

func (s *facebookStore) Unbind(ctx context.Context, gateID string) error {
	const query = `DELETE FROM im_provider.gate_facebook WHERE gate_id = $1`
	res, err := s.pool.Exec(ctx, query, gateID)
	if err != nil {
		return fmt.Errorf("postgres: unbind facebook gate: %w", err)
	}
	if res.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *facebookStore) mapVirtualFields(g *model.FacebookGate) {
	// 1. Mirror name
	g.PageName = g.Name

	// 2. Map status from boolean
	if g.Enabled {
		g.Status = model.StatusActive
	} else {
		g.Status = model.StatusDisabled
	}
}
