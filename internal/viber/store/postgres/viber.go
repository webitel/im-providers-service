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
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
	vibstore "github.com/webitel/im-providers-service/internal/viber/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var _ vibstore.ViberStore = (*viberStore)(nil)

type viberStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
	cache  sharedstore.GateCache
}

func NewViberStore(pool *pgxpool.Pool, crypt crypto.Encryptor, cache sharedstore.GateCache) vibstore.ViberStore {
	return &viberStore{pool: pool, crypto: crypt, cache: cache}
}

const selectColumns = `
	g.id,
	g.dc AS domain_id,
	g.name,
	g.enabled,
	g.created_at,
	g.updated_at,
	b.sub AS "peer.sub",
	b.iss AS "peer.iss",
	vb.bot_id,
	COALESCE(vb.bot_uri, '') AS bot_uri,
	vb.auth_token,
	vb.sender_name,
	COALESCE(vb.sender_avatar, '') AS sender_avatar,
	vb.webhook_uri`

func (s *viberStore) Insert(ctx context.Context, dc int64, g *vibmodel.ViberGate) error {
	token, err := s.crypto.Encrypt(g.AuthToken)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	const query = `
	WITH new_gate AS (
		INSERT INTO im_provider.gates (dc, name, type, enabled)
		VALUES ($1, $2, 'viber', $3)
		RETURNING id, name, created_at, updated_at
	),
	new_bot AS (
		INSERT INTO im_provider.bots (sub, iss, gate_id)
		SELECT $4, $5, id FROM new_gate
		RETURNING id
	)
	INSERT INTO im_provider.gate_viber (gate_id, bot_id, bot_uri, auth_token, sender_name, sender_avatar, webhook_uri)
	SELECT id, $6, $7, $8, $9, $10, $11 FROM new_gate
	RETURNING
		gate_id,
		(SELECT name FROM new_gate),
		(SELECT created_at FROM new_gate),
		(SELECT updated_at FROM new_gate)`

	err = s.pool.QueryRow(ctx, query,
		dc, g.Name, g.Enabled,
		g.Peer.Sub, g.Peer.Iss,
		g.BotID, g.BotURI, token, g.SenderName, g.SenderAvatar, g.WebhookURI,
	).Scan(&g.ID, &g.Name, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: insert viber gateway: %w", err)
	}

	g.DomainID = dc
	s.mapVirtualFields(g)
	return nil
}

func (s *viberStore) Select(ctx context.Context, id string) (*vibmodel.ViberGate, error) {
	query := `SELECT ` + selectColumns + `
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.gate_viber vb ON g.id = vb.gate_id
	WHERE g.id = $1`

	return s.selectOne(ctx, query, id)
}

func (s *viberStore) SelectByURI(ctx context.Context, uri string) (*vibmodel.ViberGate, error) {
	query := `SELECT ` + selectColumns + `
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.gate_viber vb ON g.id = vb.gate_id
	WHERE vb.webhook_uri = $1`

	return s.selectOne(ctx, query, uri)
}

func (s *viberStore) selectOne(ctx context.Context, query, arg string) (*vibmodel.ViberGate, error) {
	var g vibmodel.ViberGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, arg); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}
		return nil, fmt.Errorf("postgres: select viber gate: %w", err)
	}

	if dec, err := s.crypto.Decrypt(g.AuthToken); err == nil {
		g.AuthToken = dec
	}

	s.mapVirtualFields(&g)
	return &g, nil
}

func (s *viberStore) Update(ctx context.Context, g *vibmodel.ViberGate) error {
	token, err := s.crypto.Encrypt(g.AuthToken)
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
			UPDATE im_provider.gate_viber
			SET auth_token = $1, sender_name = $2, sender_avatar = $3
			WHERE gate_id = $4`
		_, err := tx.Exec(ctx, uConfig, token, g.SenderName, g.SenderAvatar, g.ID)
		return err
	})

	if err == nil && g.WebhookURI != "" {
		s.cache.Delete(g.WebhookURI)
	}
	return err
}

func (s *viberStore) Unbind(ctx context.Context, gateID string) error {
	var webhookURI string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		scanErr := tx.QueryRow(ctx,
			"SELECT webhook_uri FROM im_provider.gate_viber WHERE gate_id = $1", gateID,
		).Scan(&webhookURI)
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
	if webhookURI != "" {
		s.cache.Delete(webhookURI)
	}
	return nil
}

func (s *viberStore) mapVirtualFields(g *vibmodel.ViberGate) {
	if g.Enabled {
		g.Status = sharedmodel.StatusActive
	} else {
		g.Status = sharedmodel.StatusDisabled
	}
}
