package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

const (
	telegramGateType = model.ProviderType
)

func NewTelegramBotStore(pool *pgxpool.Pool, crypt crypto.Encryptor) *TelegramBotStore {
	return &TelegramBotStore{
		pool:   pool,
		crypto: crypt,
	}
}

type TelegramBotStore struct {
	pool   *pgxpool.Pool
	crypto crypto.Encryptor
}

type gateResult struct {
	ID            uuid.UUID
	DC            int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Name          string
	Enabled       bool
	BotSub        string
	BotIss        string
	Token         string
	URI           string
	WebhookSecret string
}

func (s *TelegramBotStore) Insert(ctx context.Context, gate *model.CreateGate) (*model.Gate, error) {
	query := `
		WITH gate AS (
			INSERT INTO im_provider.gates(dc, name, type, enabled)
			VALUES (@DC, @Name, @Type, @Enabled)
			RETURNING id, dc, created_at, updated_at, name, enabled
		),
		bot AS (
            INSERT INTO im_provider.bots(sub, iss, gate_id)
            SELECT @BotSub, @BotIss, id FROM gate
            RETURNING gate_id, sub, iss
		),
		telegram AS (
			INSERT INTO im_provider.telegram(gate_id, token, uri, webhook_secret)
			SELECT id, @Token, @URI, @WebhookSecret FROM gate
			RETURNING gate_id, token, uri, webhook_secret
		)

		SELECT
			gate.id,
			gate.dc,
			gate.created_at,
			gate.updated_at,
			gate.name,
			gate.enabled,
			bot.sub,
			bot.iss,
			telegram.token,
			telegram.uri,
			telegram.webhook_secret
		FROM gate
		JOIN bot ON gate.id = bot.gate_id
		JOIN telegram ON gate.id = telegram.gate_id
		`
	token, err := s.crypto.Encrypt(gate.Token)
	if err != nil {
		return nil, errors.Internal("failed to encrypt token")
	}
	webhookSecret, err := s.crypto.Encrypt(gate.WebhookSecret)
	if err != nil {
		return nil, errors.Internal("failed to encrypt webhook secret")
	}

	row := s.pool.QueryRow(ctx, query,
		pgx.NamedArgs{
			"DC":            gate.DC,
			"Name":          gate.Name,
			"Type":          telegramGateType,
			"Enabled":       gate.Enabled,
			"BotSub":        gate.Bot.Sub,
			"BotIss":        gate.Bot.Iss,
			"Token":         token,
			"URI":           gate.URI,
			"WebhookSecret": webhookSecret,
		},
	)
	return s.scanRow(row)
}

func (s *TelegramBotStore) GetByURI(ctx context.Context, uri string) (*model.Gate, error) {
	query := `
		SELECT
			g.id,
			g.dc,
			g.created_at,
			g.updated_at,
			g.name,
			g.enabled,
			b.sub,
			b.iss,
			t.token,
			t.uri,
			t.webhook_secret
		FROM im_provider.gates g
		JOIN im_provider.bots b ON g.id = b.gate_id
		JOIN im_provider.telegram t ON g.id = t.gate_id
		WHERE t.uri = $1 AND g.type = $2
	`
	row := s.pool.QueryRow(ctx, query, uri, telegramGateType)
	return s.scanRow(row)
}

func (s *TelegramBotStore) Get(ctx context.Context, filters *model.GetGateFilters) (*model.Gate, error) {
	if filters == nil {
		return nil, errors.Internal("filters required")
	}

	query := `
		SELECT
			g.id,
			g.dc,
			g.created_at,
			g.updated_at,
			g.name,
			g.enabled,
			b.sub,
			b.iss,
			t.token,
			t.uri,
			t.webhook_secret
		FROM im_provider.gates g
		JOIN im_provider.bots b ON g.id = b.gate_id
		JOIN im_provider.telegram t ON g.id = t.gate_id
		WHERE (
		   @ID::uuid IS NULL OR g.id = @ID
		) AND (
		   @DC::bigint IS NULL OR g.dc = @DC
		) AND (
		   @URI::text IS NULL OR t.uri = @URI
		) AND g.type = @Type
	`
	row := s.pool.QueryRow(ctx, query, pgx.NamedArgs{
		"ID":   filters.ID,
		"DC":   filters.DC,
		"URI":  filters.URI,
		"Type": telegramGateType,
	})
	return s.scanRow(row)
}

func (s *TelegramBotStore) Update(ctx context.Context, gate *model.UpdateGate) (*model.Gate, error) {
	var (
		gateQuery = `
		WITH updated_gate AS (
			UPDATE im_provider.gates
			SET name = coalesce(@Name, name), enabled = coalesce(@Enabled, enabled)
			WHERE id = @ID AND dc = @DC AND type = @Type
			RETURNING id, dc, created_at, updated_at, name, enabled
		),
		updated_bot AS (
			UPDATE im_provider.bots b
			SET sub = coalesce(@Sub, sub), iss = coalesce(@Iss, iss)
			FROM updated_gate g
			WHERE b.gate_id = g.id
			RETURNING b.gate_id, b.sub, b.iss
		),
		updated_telegram AS (
			UPDATE im_provider.telegram t
			SET token = coalesce(@Token, token),
			    webhook_secret = coalesce(@WebhookSecret, webhook_secret),
			    updated_at = now()
			FROM updated_gate g
			WHERE t.gate_id = g.id
			RETURNING t.gate_id, t.token, t.uri, t.webhook_secret
		)
		SELECT g.id, g.dc, g.created_at, g.updated_at, g.name, g.enabled, b.sub, b.iss, t.token, t.uri, t.webhook_secret
		FROM updated_gate g
		JOIN updated_bot b ON b.gate_id = g.id
		JOIN updated_telegram t ON t.gate_id = g.id
	`

		token         *string
		webhookSecret *string
		sub           *string
		iss           *string
	)
	if gate.Bot != nil {
		sub, iss = &gate.Bot.Sub, &gate.Bot.Iss
	}
	if gate.Token != nil {
		encryptedToken, err := s.crypto.Encrypt(*gate.Token)
		if err != nil {
			return nil, errors.Internal("failed to encrypt token", errors.WithCause(err))
		}
		token = &encryptedToken
	}
	if gate.WebhookSecret != nil {
		encryptedSecret, err := s.crypto.Encrypt(*gate.WebhookSecret)
		if err != nil {
			return nil, errors.Internal("failed to encrypt webhook secret", errors.WithCause(err))
		}
		webhookSecret = &encryptedSecret
	}

	row := s.pool.QueryRow(ctx, gateQuery,
		pgx.NamedArgs{
			"ID":            gate.ID,
			"DC":            gate.DC,
			"Type":          telegramGateType,
			"Name":          gate.Name,
			"Enabled":       gate.Enabled,
			"Sub":           sub,
			"Iss":           iss,
			"Token":         token,
			"WebhookSecret": webhookSecret,
		},
	)
	res, err := s.scanRow(row)
	if err != nil {
		return nil, errors.Internal("failed to scan result", errors.WithCause(err))
	}

	return res, nil
}

func (s *TelegramBotStore) scanRow(row pgx.Row) (*model.Gate, error) {
	var res gateResult
	err := row.Scan(
		&res.ID,
		&res.DC,
		&res.CreatedAt,
		&res.UpdatedAt,
		&res.Name,
		&res.Enabled,
		&res.BotSub,
		&res.BotIss,
		&res.Token,
		&res.URI,
		&res.WebhookSecret,
	)
	if err != nil {
		return nil, err
	}
	token, err := s.crypto.Decrypt(res.Token)
	if err != nil {
		return nil, err
	}
	webhookSecret, err := s.crypto.Decrypt(res.WebhookSecret)
	if err != nil {
		return nil, err
	}

	return &model.Gate{
		ID:            res.ID,
		DC:            res.DC,
		CreatedAt:     res.CreatedAt,
		UpdatedAt:     res.UpdatedAt,
		Name:          res.Name,
		Enabled:       res.Enabled,
		Bot:           &coremodel.Peer{Sub: res.BotSub, Iss: res.BotIss},
		Token:         token,
		URI:           res.URI,
		WebhookSecret: webhookSecret,
	}, nil
}

func (s *TelegramBotStore) Delete(ctx context.Context, id uuid.UUID, domainID int64) error {
	query := `DELETE FROM im_provider.gates WHERE id = $1 AND dc = $2 AND type = $3`
	_, err := s.pool.Exec(ctx, query, id, domainID, telegramGateType)
	if err != nil {
		return errors.Internal("delete telegram bot store error", errors.WithCause(err))
	}
	return nil
}
