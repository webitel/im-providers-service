package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
	igstore "github.com/webitel/im-providers-service/internal/instagram/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var _ igstore.InstagramStore = (*instagramStore)(nil)

type instagramStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
	cache  sharedstore.GateCache
}

func NewInstagramStore(pool *pgxpool.Pool, crypt crypto.Encryptor, cache sharedstore.GateCache) igstore.InstagramStore {
	return &instagramStore{
		pool:   pool,
		crypto: crypt,
		cache:  cache,
	}
}

func (s *instagramStore) Insert(ctx context.Context, dc int64, g *igmodel.InstagramGate) error {
	token, err := s.crypto.Encrypt(g.IGToken)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	const query = `
	WITH new_gate AS (
		INSERT INTO im_provider.gates (dc, name, type, enabled)
		VALUES ($1, $2, 'instagram', $3)
		RETURNING id, name, created_at, updated_at
	),
	new_bot AS (
		INSERT INTO im_provider.bots (sub, iss, gate_id)
		SELECT $4, $5, id FROM new_gate
		RETURNING id
	)
	INSERT INTO im_provider.instagram (gate_id, meta_app_id, business_account_id, ig_access_token)
	SELECT id, $6, $7, $8 FROM new_gate
	RETURNING
		gate_id,
		(SELECT name FROM new_gate),
		(SELECT created_at FROM new_gate),
		(SELECT updated_at FROM new_gate)`

	err = s.pool.QueryRow(ctx, query,
		dc, g.Name, g.Enabled, g.Peer.Sub, g.Peer.Iss, g.MetaAppID, g.BusinessAccountID, token,
	).Scan(&g.ID, &g.Name, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: insert instagram gateway: %w", err)
	}

	s.mapVirtualFields(g)

	return nil
}

func (s *instagramStore) Select(ctx context.Context, id string) (*igmodel.InstagramGate, error) {
	const query = `
	SELECT
		g.id, g.dc AS domain_id, g.name, g.enabled, g.created_at, g.updated_at,
		b.sub AS "peer.sub", b.iss AS "peer.iss",
		ig.meta_app_id, ig.business_account_id, ig.ig_access_token
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.instagram ig ON g.id = ig.gate_id
	WHERE g.id = $1`

	var g igmodel.InstagramGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, id); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}

		return nil, fmt.Errorf("postgres: select instagram gate: %w", err)
	}

	if dec, err := s.crypto.Decrypt(g.IGToken); err == nil {
		g.IGToken = dec
	}

	s.mapVirtualFields(&g)

	return &g, nil
}

func (s *instagramStore) SelectByBusinessAccountAndURI(ctx context.Context, businessAccountID, uri string) (*igmodel.InstagramGate, error) {
	const query = `
	SELECT
		g.id,
		g.dc AS domain_id,
		g.name,
		g.enabled,
		g.created_at,
		g.updated_at,
		b.sub AS "peer.sub",
		b.iss AS "peer.iss",
		ig.meta_app_id,
		ig.business_account_id,
		ig.ig_access_token
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.instagram ig ON g.id = ig.gate_id
	JOIN im_provider.meta_apps ma ON ig.meta_app_id = ma.id
	WHERE ig.business_account_id = $1 AND ma.uri = $2`

	var g igmodel.InstagramGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, businessAccountID, uri); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}

		return nil, fmt.Errorf("postgres: select by business_account_id and uri: %w", err)
	}

	if dec, err := s.crypto.Decrypt(g.IGToken); err == nil {
		g.IGToken = dec
	}

	s.mapVirtualFields(&g)

	return &g, nil
}

func (s *instagramStore) Update(ctx context.Context, g *igmodel.InstagramGate) error {
	token, err := s.crypto.Encrypt(g.IGToken)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		const uGate = `UPDATE im_provider.gates SET name = $1, enabled = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`
		if err := tx.QueryRow(ctx, uGate, g.Name, g.Enabled, g.ID).Scan(&g.UpdatedAt); err != nil {
			return err
		}

		const uBot = `UPDATE im_provider.bots SET sub = $1, iss = $2 WHERE gate_id = $3`
		if _, err := tx.Exec(ctx, uBot, g.Peer.Sub, g.Peer.Iss, g.ID); err != nil {
			return err
		}

		const uConfig = `
			UPDATE im_provider.instagram
			SET meta_app_id = $1, business_account_id = $2, ig_access_token = $3
			WHERE gate_id = $4`

		_, err := tx.Exec(ctx, uConfig, g.MetaAppID, g.BusinessAccountID, token, g.ID)

		return err
	})
	if err == nil {
		s.cache.Delete(igstore.GateCacheKey(g.BusinessAccountID))
	}

	return err
}

func (s *instagramStore) Unbind(ctx context.Context, gateID string) error {
	var businessAccountID string

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		scanErr := tx.QueryRow(ctx,
			"SELECT business_account_id FROM im_provider.instagram WHERE gate_id = $1", gateID,
		).Scan(&businessAccountID)
		if scanErr != nil && !errors.Is(scanErr, pgx.ErrNoRows) {
			return scanErr
		}

		res, execErr := tx.Exec(ctx, "DELETE FROM im_provider.gates WHERE id = $1", gateID)
		if execErr != nil {
			return fmt.Errorf("postgres: delete gate: %w", execErr)
		}

		if res.RowsAffected() == 0 {
			return sharedstore.ErrNotFound
		}

		return nil
	})
	if err != nil {
		return err
	}

	if businessAccountID != "" {
		s.cache.Delete(igstore.GateCacheKey(businessAccountID))
	}

	return nil
}

func (s *instagramStore) mapVirtualFields(g *igmodel.InstagramGate) {
	if g.Enabled {
		g.Status = sharedmodel.StatusActive
	} else {
		g.Status = sharedmodel.StatusDisabled
	}
}
