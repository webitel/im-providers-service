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
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var _ customstore.CustomStore = (*customStore)(nil)

type customStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
	cache  sharedstore.GateCache
}

func NewCustomStore(pool *pgxpool.Pool, crypt crypto.Encryptor, cache sharedstore.GateCache) customstore.CustomStore {
	return &customStore{pool: pool, crypto: crypt, cache: cache}
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
	cu.callback_url,
	cu.app_secret,
	cu.webhook_uri,
	cu.allowed_ips,
	cu.request_timeout_ms,
	cu.retry_attempts`

func (s *customStore) Insert(ctx context.Context, dc int64, g *custommodel.CustomGate) error {
	secret, err := s.crypto.Encrypt(g.AppSecret)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}

	const query = `
	WITH new_gate AS (
		INSERT INTO im_provider.gates (dc, name, type, enabled)
		VALUES ($1, $2, 'custom', $3)
		RETURNING id, name, created_at, updated_at
	),
	new_bot AS (
		INSERT INTO im_provider.bots (sub, iss, gate_id)
		SELECT $4, $5, id FROM new_gate
		RETURNING id
	)
	INSERT INTO im_provider.gate_custom (gate_id, callback_url, app_secret, webhook_uri, allowed_ips, request_timeout_ms, retry_attempts)
	SELECT id, $6, $7, $8, $9, $10, $11 FROM new_gate
	RETURNING
		gate_id,
		(SELECT name FROM new_gate),
		(SELECT created_at FROM new_gate),
		(SELECT updated_at FROM new_gate)`

	err = s.pool.QueryRow(ctx, query,
		dc, g.Name, g.Enabled,
		g.Peer.Sub, g.Peer.Iss,
		g.CallbackURL, secret, g.WebhookURI, allowedIPsArg(g.AllowedIPs), g.RequestTimeoutMS, g.RetryAttempts,
	).Scan(&g.ID, &g.Name, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: insert custom gateway: %w", err)
	}

	g.DomainID = dc
	s.mapVirtualFields(g)

	return nil
}

func (s *customStore) Select(ctx context.Context, id string) (*custommodel.CustomGate, error) {
	query := `SELECT ` + selectColumns + `
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.gate_custom cu ON g.id = cu.gate_id
	WHERE g.id = $1`

	return s.selectOne(ctx, query, id)
}

func (s *customStore) SelectByURI(ctx context.Context, uri string) (*custommodel.CustomGate, error) {
	query := `SELECT ` + selectColumns + `
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.gate_custom cu ON g.id = cu.gate_id
	WHERE cu.webhook_uri = $1`

	return s.selectOne(ctx, query, uri)
}

func (s *customStore) selectOne(ctx context.Context, query, arg string) (*custommodel.CustomGate, error) {
	var g custommodel.CustomGate
	if err := pgxscan.Get(ctx, s.pool, &g, query, arg); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}

		return nil, fmt.Errorf("postgres: select custom gate: %w", err)
	}

	if dec, err := s.crypto.Decrypt(g.AppSecret); err == nil {
		g.AppSecret = dec
	}

	s.mapVirtualFields(&g)

	return &g, nil
}

func (s *customStore) Update(ctx context.Context, g *custommodel.CustomGate) error {
	secret, err := s.crypto.Encrypt(g.AppSecret)
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
			UPDATE im_provider.gate_custom
			SET callback_url = $1, app_secret = $2, allowed_ips = $3, request_timeout_ms = $4, retry_attempts = $5
			WHERE gate_id = $6`

		_, err := tx.Exec(ctx, uConfig,
			g.CallbackURL, secret, allowedIPsArg(g.AllowedIPs), g.RequestTimeoutMS, g.RetryAttempts, g.ID,
		)

		return err
	})

	if err == nil && g.WebhookURI != "" {
		s.cache.Delete(g.WebhookURI)
	}

	return err
}

func (s *customStore) Unbind(ctx context.Context, gateID string) error {
	var webhookURI string

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		scanErr := tx.QueryRow(ctx,
			"SELECT webhook_uri FROM im_provider.gate_custom WHERE gate_id = $1", gateID,
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

func (s *customStore) UpsertChat(ctx context.Context, gateID, chatID, externalSub string) error {
	// A returning user starts a new chatId while the old row still points at
	// them; the unique key on (gate_id, external_sub) forces the older
	// conversation out so outbound always resolves the current one.
	const query = `
		WITH superseded AS (
			DELETE FROM im_provider.gate_custom_chats
			WHERE gate_id = $1 AND external_sub = $2 AND chat_id <> $3
		)
		INSERT INTO im_provider.gate_custom_chats (gate_id, chat_id, external_sub)
		VALUES ($1, $3, $2)
		ON CONFLICT (gate_id, chat_id) DO UPDATE
		SET external_sub = EXCLUDED.external_sub, closed_at = NULL, updated_at = NOW()`

	if _, err := s.pool.Exec(ctx, query, gateID, externalSub, chatID); err != nil {
		return fmt.Errorf("postgres: upsert custom chat: %w", err)
	}

	return nil
}

func (s *customStore) ChatByRecipient(ctx context.Context, gateID, externalSub string) (*custommodel.Chat, error) {
	const query = `
		SELECT gate_id, chat_id, external_sub, closed_at
		FROM im_provider.gate_custom_chats
		WHERE gate_id = $1 AND external_sub = $2 AND closed_at IS NULL`

	return s.selectChat(ctx, query, gateID, externalSub)
}

func (s *customStore) ChatByID(ctx context.Context, gateID, chatID string) (*custommodel.Chat, error) {
	const query = `
		SELECT gate_id, chat_id, external_sub, closed_at
		FROM im_provider.gate_custom_chats
		WHERE gate_id = $1 AND chat_id = $2`

	return s.selectChat(ctx, query, gateID, chatID)
}

func (s *customStore) selectChat(ctx context.Context, query, gateID, arg string) (*custommodel.Chat, error) {
	var chat custommodel.Chat
	if err := pgxscan.Get(ctx, s.pool, &chat, query, gateID, arg); err != nil {
		if pgxscan.NotFound(err) {
			return nil, sharedstore.ErrNotFound
		}

		return nil, fmt.Errorf("postgres: select custom chat: %w", err)
	}

	return &chat, nil
}

func (s *customStore) mapVirtualFields(g *custommodel.CustomGate) {
	if g.Enabled {
		g.Status = sharedmodel.StatusActive
	} else {
		g.Status = sharedmodel.StatusDisabled
	}
}

func allowedIPsArg(entries []string) []string {
	if entries == nil {
		return make([]string, 0)
	}

	return entries
}
