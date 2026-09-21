package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

var _ customstore.CustomStore = (*CustomStore)(nil)

type CustomStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

func NewCustomStore(pool *pgxpool.Pool, crypt crypto.Encryptor) *CustomStore {
	return &CustomStore{pool: pool, crypto: crypt}
}

type gateRow struct {
	ID               string    `db:"id"`
	DomainID         int64     `db:"domain_id"`
	Name             string    `db:"name"`
	Enabled          bool      `db:"enabled"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
	PeerSub          string    `db:"peer_sub"`
	PeerIss          string    `db:"peer_iss"`
	CallbackURL      string    `db:"callback_url"`
	AppSecret        string    `db:"app_secret"`
	WebhookURI       string    `db:"webhook_uri"`
	AllowedIPs       []string  `db:"allowed_ips"`
	RequestTimeoutMS int32     `db:"request_timeout_ms"`
	RetryAttempts    int32     `db:"retry_attempts"`
}

const selectColumns = `
	g.id,
	g.dc AS domain_id,
	g.name,
	g.enabled,
	g.created_at,
	g.updated_at,
	b.sub AS peer_sub,
	b.iss AS peer_iss,
	c.callback_url,
	c.app_secret,
	c.webhook_uri,
	c.allowed_ips,
	c.request_timeout_ms,
	c.retry_attempts`

func (s *CustomStore) Insert(ctx context.Context, dc int64, g *custommodel.CustomGate) error {
	secret, err := s.crypto.Encrypt(g.AppSecret)
	if err != nil {
		return errors.Internal("failed to encrypt app secret",
			errors.WithCause(err), errors.WithID("custom.store.postgres.insert"))
	}

	const query = `
	WITH new_gate AS (
		INSERT INTO im_provider.gates (dc, name, type, enabled)
		VALUES (@DomainID, @Name, @Type, @Enabled)
		RETURNING id, dc, name, enabled, created_at, updated_at
	),
	new_bot AS (
		INSERT INTO im_provider.bots (sub, iss, gate_id)
		SELECT @PeerSub, @PeerIss, id FROM new_gate
		RETURNING gate_id, sub, iss
	),
	new_config AS (
		INSERT INTO im_provider.gate_custom (gate_id, callback_url, app_secret, webhook_uri, allowed_ips, request_timeout_ms, retry_attempts)
		SELECT id, @CallbackURL, @AppSecret, @WebhookURI, @AllowedIPs, @RequestTimeoutMS, @RetryAttempts FROM new_gate
		RETURNING gate_id, callback_url, app_secret, webhook_uri, allowed_ips, request_timeout_ms, retry_attempts
	)
	SELECT ` + selectColumns + `
	FROM new_gate g
	JOIN new_bot b ON b.gate_id = g.id
	JOIN new_config c ON c.gate_id = g.id`

	row, err := s.queryGate(ctx, query, pgx.NamedArgs{
		"DomainID":         dc,
		"Name":             g.Name,
		"Type":             custommodel.ProviderType,
		"Enabled":          g.Enabled,
		"PeerSub":          g.Peer.Sub,
		"PeerIss":          g.Peer.Iss,
		"CallbackURL":      g.CallbackURL,
		"AppSecret":        secret,
		"WebhookURI":       g.WebhookURI,
		"AllowedIPs":       allowedIPsArg(g.AllowedIPs),
		"RequestTimeoutMS": g.RequestTimeoutMS,
		"RetryAttempts":    g.RetryAttempts,
	}, "custom.store.postgres.insert")
	if err != nil {
		return err
	}

	return s.fillGate(row, g)
}

func (s *CustomStore) Select(ctx context.Context, filter custommodel.GateFilter) (*custommodel.CustomGate, error) {
	const errID = "custom.store.postgres.select"

	if filter.ID == nil && filter.WebhookURI == nil {
		return nil, errors.InvalidArgument("custom: gate filter is empty", errors.WithID(errID))
	}

	const query = `SELECT ` + selectColumns + `
	FROM im_provider.gates g
	JOIN im_provider.bots b ON g.id = b.gate_id
	JOIN im_provider.gate_custom c ON g.id = c.gate_id
	WHERE (@ID::uuid IS NULL OR g.id = @ID)
	  AND (@WebhookURI::text IS NULL OR c.webhook_uri = @WebhookURI)`

	row, err := s.queryGate(ctx, query, pgx.NamedArgs{
		"ID":         filter.ID,
		"WebhookURI": filter.WebhookURI,
	}, errID)
	if err != nil {
		return nil, err
	}

	gate := &custommodel.CustomGate{}
	if err := s.fillGate(row, gate); err != nil {
		return nil, err
	}

	return gate, nil
}

func (s *CustomStore) Update(ctx context.Context, req custommodel.UpdateCustom) (*custommodel.CustomGate, error) {
	const errID = "custom.store.postgres.update"

	var secret *string

	if req.AppSecret != nil {
		encrypted, err := s.crypto.Encrypt(*req.AppSecret)
		if err != nil {
			return nil, errors.Internal("failed to encrypt app secret",
				errors.WithCause(err), errors.WithID(errID))
		}

		secret = &encrypted
	}

	var peerSub, peerIss *string

	if req.Peer != nil {
		peerSub, peerIss = &req.Peer.Sub, &req.Peer.Iss
	}

	var allowedIPs *[]string

	if req.AllowedIPs != nil {
		entries := allowedIPsArg(*req.AllowedIPs)
		allowedIPs = &entries
	}

	const query = `
	WITH updated_gate AS (
		UPDATE im_provider.gates
		SET name = COALESCE(@Name::text, name),
		    enabled = COALESCE(@Enabled::boolean, enabled)
		WHERE id = @ID AND type = @Type
		RETURNING id, dc, name, enabled, created_at, updated_at
	),
	updated_bot AS (
		UPDATE im_provider.bots b
		SET sub = COALESCE(@PeerSub::text, sub),
		    iss = COALESCE(@PeerIss::text, iss)
		FROM updated_gate g
		WHERE b.gate_id = g.id
		RETURNING b.gate_id, b.sub, b.iss
	),
	updated_config AS (
		UPDATE im_provider.gate_custom c
		SET callback_url = COALESCE(@CallbackURL::text, callback_url),
		    app_secret = COALESCE(@AppSecret::text, app_secret),
		    allowed_ips = COALESCE(@AllowedIPs::text[], allowed_ips),
		    request_timeout_ms = COALESCE(@RequestTimeoutMS::integer, request_timeout_ms),
		    retry_attempts = COALESCE(@RetryAttempts::integer, retry_attempts)
		FROM updated_gate g
		WHERE c.gate_id = g.id
		RETURNING c.gate_id, c.callback_url, c.app_secret, c.webhook_uri, c.allowed_ips, c.request_timeout_ms, c.retry_attempts
	)
	SELECT ` + selectColumns + `
	FROM updated_gate g
	JOIN updated_bot b ON b.gate_id = g.id
	JOIN updated_config c ON c.gate_id = g.id`

	row, err := s.queryGate(ctx, query, pgx.NamedArgs{
		"ID":               req.ID,
		"Type":             custommodel.ProviderType,
		"Name":             req.Name,
		"Enabled":          req.Enabled,
		"PeerSub":          peerSub,
		"PeerIss":          peerIss,
		"CallbackURL":      req.CallbackURL,
		"AppSecret":        secret,
		"AllowedIPs":       allowedIPs,
		"RequestTimeoutMS": req.RequestTimeoutMS,
		"RetryAttempts":    req.RetryAttempts,
	}, errID)
	if err != nil {
		return nil, err
	}

	gate := &custommodel.CustomGate{}
	if err := s.fillGate(row, gate); err != nil {
		return nil, err
	}

	return gate, nil
}

func (s *CustomStore) Unbind(ctx context.Context, gateID string) (*custommodel.CustomGate, error) {
	const query = `
	WITH target AS (
		SELECT ` + selectColumns + `
		FROM im_provider.gates g
		JOIN im_provider.bots b ON g.id = b.gate_id
		JOIN im_provider.gate_custom c ON g.id = c.gate_id
		WHERE g.id = @ID AND g.type = @Type
	),
	deleted AS (
		DELETE FROM im_provider.gates WHERE id IN (SELECT id FROM target)
	)
	SELECT * FROM target`

	row, err := s.queryGate(ctx, query, pgx.NamedArgs{
		"ID":   gateID,
		"Type": custommodel.ProviderType,
	}, "custom.store.postgres.unbind")
	if err != nil {
		return nil, err
	}

	gate := &custommodel.CustomGate{}
	if err := s.fillGate(row, gate); err != nil {
		return nil, err
	}

	return gate, nil
}

func (s *CustomStore) UpsertChat(ctx context.Context, gateID, chatID, externalSub string) error {
	// A returning user starts a new chatId while the old row still points at
	// them; the unique key on (gate_id, external_sub) forces the older
	// conversation out so outbound always resolves the current one.
	const query = `
		WITH superseded AS (
			DELETE FROM im_provider.gate_custom_chats
			WHERE gate_id = @GateID AND external_sub = @ExternalSub AND chat_id <> @ChatID
		)
		INSERT INTO im_provider.gate_custom_chats (gate_id, chat_id, external_sub)
		VALUES (@GateID, @ChatID, @ExternalSub)
		ON CONFLICT (gate_id, chat_id) DO UPDATE
		SET external_sub = EXCLUDED.external_sub, closed_at = NULL, updated_at = NOW()`

	_, err := s.pool.Exec(ctx, query, pgx.NamedArgs{
		"GateID":      gateID,
		"ChatID":      chatID,
		"ExternalSub": externalSub,
	})
	if err != nil {
		return errors.Internal("failed to upsert custom chat",
			errors.WithCause(err), errors.WithID("custom.store.postgres.upsert_chat"))
	}

	return nil
}

func (s *CustomStore) ChatByRecipient(ctx context.Context, gateID, externalSub string) (*custommodel.Chat, error) {
	return s.selectChat(ctx, gateID, &externalSub, nil, "custom.store.postgres.chat_by_recipient")
}

func (s *CustomStore) ChatByID(ctx context.Context, gateID, chatID string) (*custommodel.Chat, error) {
	return s.selectChat(ctx, gateID, nil, &chatID, "custom.store.postgres.chat_by_id")
}

func (s *CustomStore) selectChat(ctx context.Context, gateID string, externalSub, chatID *string, errID string) (*custommodel.Chat, error) {
	const query = `
		SELECT gate_id, chat_id, external_sub, closed_at
		FROM im_provider.gate_custom_chats
		WHERE gate_id = @GateID
		  AND (@ExternalSub::text IS NULL OR external_sub = @ExternalSub)
		  AND (@ChatID::text IS NULL OR chat_id = @ChatID)
		  AND (@ChatID::text IS NOT NULL OR closed_at IS NULL)`

	rows, err := s.pool.Query(ctx, query, pgx.NamedArgs{
		"GateID":      gateID,
		"ExternalSub": externalSub,
		"ChatID":      chatID,
	})
	if err != nil {
		return nil, errors.Internal("failed to query custom chat",
			errors.WithCause(err), errors.WithID(errID))
	}

	chat, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[custommodel.Chat])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound(errID)
		}

		return nil, errors.Internal("failed to scan custom chat",
			errors.WithCause(err), errors.WithID(errID))
	}

	return chat, nil
}

func (s *CustomStore) queryGate(ctx context.Context, query string, args pgx.NamedArgs, errID string) (*gateRow, error) {
	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, errors.Internal("failed to query custom gateway",
			errors.WithCause(err), errors.WithID(errID))
	}

	row, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByName[gateRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound(errID)
		}

		return nil, errors.Internal("failed to scan custom gateway",
			errors.WithCause(err), errors.WithID(errID))
	}

	return row, nil
}

func (s *CustomStore) fillGate(row *gateRow, g *custommodel.CustomGate) error {
	secret, err := s.crypto.Decrypt(row.AppSecret)
	if err != nil {
		return errors.Internal("failed to decrypt app secret",
			errors.WithCause(err), errors.WithID("custom.store.postgres.fill_gate"))
	}

	g.ID = row.ID
	g.DomainID = row.DomainID
	g.Peer = sharedmodel.Peer{Sub: row.PeerSub, Iss: row.PeerIss}
	g.Name = row.Name
	g.CallbackURL = row.CallbackURL
	g.AppSecret = secret
	g.WebhookURI = row.WebhookURI
	g.AllowedIPs = row.AllowedIPs
	g.RequestTimeoutMS = row.RequestTimeoutMS
	g.RetryAttempts = row.RetryAttempts
	g.CreatedAt = row.CreatedAt
	g.UpdatedAt = row.UpdatedAt
	g.Enabled = row.Enabled

	if row.Enabled {
		g.Status = sharedmodel.StatusActive
	} else {
		g.Status = sharedmodel.StatusDisabled
	}

	return nil
}

func notFound(errID string) error {
	return errors.NotFound("custom gateway not found",
		errors.WithCause(sharedstore.ErrNotFound), errors.WithID(errID))
}

func allowedIPsArg(entries []string) []string {
	if entries == nil {
		return make([]string, 0)
	}

	return entries
}
