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
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
	vibbmstore "github.com/webitel/im-providers-service/internal/viberbm/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var _ vibbmstore.ViberBMStore = (*viberBMStore)(nil)

type viberBMStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
	cache  sharedstore.GateCache
}

func NewViberBMStore(pool *pgxpool.Pool, crypt crypto.Encryptor, cache sharedstore.GateCache) vibbmstore.ViberBMStore {
	return &viberBMStore{pool: pool, crypto: crypt, cache: cache}
}

// gateKey mirrors the provider's LRU key: uri + ":" + senderName.
func gateKey(uri, senderName string) string { return uri + ":" + senderName }

const selectColumns = `
	g.id,
	g.dc AS domain_id,
	g.name,
	g.enabled,
	g.created_at,
	g.updated_at,
	b.sub AS "peer.sub",
	b.iss AS "peer.iss",
	vbm.base_url,
	vbm.sender_name,
	vbm.api_key,
	COALESCE(vbm.webhook_secret, '') AS webhook_secret,
	vbm.webhook_uri`

const fromJoin = `
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.gate_viber_bm vbm ON g.id = vbm.gate_id`

func (s *viberBMStore) Insert(ctx context.Context, dc int64, g *vibbmmodel.ViberBMGate) error {
	apiKey, err := s.crypto.Encrypt(g.APIKey)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	secret, err := s.encryptSecret(g.WebhookSecret)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	const query = `
	WITH new_gate AS (
		INSERT INTO im_provider.gates (dc, name, type, enabled)
		VALUES ($1, $2, 'viber_bm', $3)
		RETURNING id, name, created_at, updated_at
	),
	new_bot AS (
		INSERT INTO im_provider.bots (sub, iss, gate_id)
		SELECT $4, $5, id FROM new_gate
		RETURNING id
	)
	INSERT INTO im_provider.gate_viber_bm (gate_id, base_url, sender_name, api_key, webhook_secret, webhook_uri)
	SELECT id, $6, $7, $8, $9, $10 FROM new_gate
	RETURNING
		gate_id,
		(SELECT name FROM new_gate),
		(SELECT created_at FROM new_gate),
		(SELECT updated_at FROM new_gate)`

	err = s.pool.QueryRow(ctx, query,
		dc, g.Name, g.Enabled,
		g.Peer.Sub, g.Peer.Iss,
		g.BaseURL, g.SenderName, apiKey, secret, g.WebhookURI,
	).Scan(&g.ID, &g.Name, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: insert viber_bm gateway: %w", err)
	}

	g.DomainID = dc
	s.mapVirtualFields(g)

	return nil
}

func (s *viberBMStore) Select(ctx context.Context, id string) (*vibbmmodel.ViberBMGate, error) {
	query := `SELECT ` + selectColumns + fromJoin + ` WHERE g.id = $1`

	return s.selectOne(ctx, query, id)
}

func (s *viberBMStore) SelectByURI(ctx context.Context, uri string) (*vibbmmodel.ViberBMGate, error) {
	query := `SELECT ` + selectColumns + fromJoin + ` WHERE vbm.webhook_uri = $1`

	return s.selectOne(ctx, query, uri)
}

func (s *viberBMStore) SelectBySenderAndURI(ctx context.Context, senderName, uri string) (*vibbmmodel.ViberBMGate, error) {
	query := `SELECT ` + selectColumns + fromJoin + ` WHERE vbm.sender_name = $1 AND vbm.webhook_uri = $2`

	var g vibbmmodel.ViberBMGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, senderName, uri); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}

		return nil, fmt.Errorf("postgres: select viber_bm gate: %w", err)
	}

	s.decryptSecrets(&g)
	s.mapVirtualFields(&g)

	return &g, nil
}

func (s *viberBMStore) List(ctx context.Context, dc int64, q string, page, size int) ([]*vibbmmodel.ViberBMGate, bool, error) {
	if size <= 0 {
		size = 20
	}

	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * size

	query := `SELECT ` + selectColumns + fromJoin + `
		WHERE g.dc = $1 AND ($2 = '' OR g.name ILIKE '%' || $2 || '%')
		ORDER BY g.created_at DESC
		LIMIT $3 OFFSET $4`

	var rows []*vibbmmodel.ViberBMGate
	if err := pgxscan.Select(ctx, s.pool, &rows, query, dc, q, size+1, offset); err != nil {
		return nil, false, fmt.Errorf("postgres: list viber_bm gates: %w", err)
	}

	next := false
	if len(rows) > size {
		next = true
		rows = rows[:size]
	}

	for _, g := range rows {
		s.decryptSecrets(g)
		s.mapVirtualFields(g)
	}

	return rows, next, nil
}

func (s *viberBMStore) selectOne(ctx context.Context, query, arg string) (*vibbmmodel.ViberBMGate, error) {
	var g vibbmmodel.ViberBMGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, arg); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}

		return nil, fmt.Errorf("postgres: select viber_bm gate: %w", err)
	}

	s.decryptSecrets(&g)
	s.mapVirtualFields(&g)

	return &g, nil
}

func (s *viberBMStore) Update(ctx context.Context, g *vibbmmodel.ViberBMGate) error {
	apiKey, err := s.crypto.Encrypt(g.APIKey)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	secret, err := s.encryptSecret(g.WebhookSecret)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	var oldSenderName string

	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			"SELECT sender_name FROM im_provider.gate_viber_bm WHERE gate_id = $1", g.ID,
		).Scan(&oldSenderName); err != nil {
			return err
		}

		const uGate = `UPDATE im_provider.gates SET name = $1, enabled = $2, updated_at = NOW() WHERE id = $3 RETURNING updated_at`
		if err := tx.QueryRow(ctx, uGate, g.Name, g.Enabled, g.ID).Scan(&g.UpdatedAt); err != nil {
			return err
		}

		const uBot = `UPDATE im_provider.bots SET sub = $1, iss = $2 WHERE gate_id = $3`
		if _, err := tx.Exec(ctx, uBot, g.Peer.Sub, g.Peer.Iss, g.ID); err != nil {
			return err
		}

		const uConfig = `
			UPDATE im_provider.gate_viber_bm
			SET base_url = $1, sender_name = $2, api_key = $3, webhook_secret = $4
			WHERE gate_id = $5`

		_, err := tx.Exec(ctx, uConfig, g.BaseURL, g.SenderName, apiKey, secret, g.ID)

		return err
	})
	if err == nil && g.WebhookURI != "" {
		s.cache.Delete(gateKey(g.WebhookURI, oldSenderName))
		s.cache.Delete(gateKey(g.WebhookURI, g.SenderName))
	}

	return err
}

func (s *viberBMStore) Unbind(ctx context.Context, gateID string) error {
	var (
		webhookURI string
		senderName string
	)

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		scanErr := tx.QueryRow(ctx,
			"SELECT webhook_uri, sender_name FROM im_provider.gate_viber_bm WHERE gate_id = $1", gateID,
		).Scan(&webhookURI, &senderName)
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
		s.cache.Delete(gateKey(webhookURI, senderName))
	}

	return nil
}

func (s *viberBMStore) encryptSecret(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}

	return s.crypto.Encrypt(plain)
}

func (s *viberBMStore) decryptSecrets(g *vibbmmodel.ViberBMGate) {
	if dec, err := s.crypto.Decrypt(g.APIKey); err == nil {
		g.APIKey = dec
	}

	if g.WebhookSecret == "" {
		return
	}

	if dec, err := s.crypto.Decrypt(g.WebhookSecret); err == nil {
		g.WebhookSecret = dec
	}
}

func (s *viberBMStore) mapVirtualFields(g *vibbmmodel.ViberBMGate) {
	if g.Enabled {
		g.Status = sharedmodel.StatusActive
	} else {
		g.Status = sharedmodel.StatusDisabled
	}
}
