package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

var _ MessageRefStore = (*messageRefStore)(nil)

type messageRefStore struct {
	pool *pgxpool.Pool
}

func NewMessageRefStore(pool *pgxpool.Pool) MessageRefStore {
	return &messageRefStore{pool: pool}
}

// Save persists the provider_message_id mapping. Duplicate webhook retries
// and provider redeliveries make conflicts possible — first write wins.
func (s *messageRefStore) Save(ctx context.Context, ref *sharedmodel.MessageRef) error {
	const q = `
		INSERT INTO im_provider.message_refs
			(gate_id, provider_message_id, provider_user_id, message_id, thread_id, member_id, domain_id, sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (gate_id, provider_message_id) DO NOTHING`

	_, err := s.pool.Exec(ctx, q,
		ref.GateID,
		ref.ProviderMessageID,
		ref.ProviderUserID,
		ref.MessageID,
		ref.ThreadID,
		ref.MemberID,
		ref.DomainID,
	)

	return err
}

// GetByProviderMessageIDs resolves refs for concrete provider message ids
// (WhatsApp statuses, Facebook delivery mids).
func (s *messageRefStore) GetByProviderMessageIDs(ctx context.Context, gateID string, providerMessageIDs []string) ([]*sharedmodel.MessageRef, error) {
	if len(providerMessageIDs) == 0 {
		return nil, nil
	}

	const q = `
		SELECT gate_id, provider_message_id, provider_user_id, message_id, thread_id, member_id, domain_id, sent_at
		  FROM im_provider.message_refs
		 WHERE gate_id = $1
		   AND provider_message_id = ANY($2)`

	rows, err := s.pool.Query(ctx, q, gateID, providerMessageIDs)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[sharedmodel.MessageRef])
}

// GetByUserUpTo returns refs of messages sent to the given platform user
// before (or at) the watermark — most recent first. Used for Facebook
// watermark-based delivery/read receipts. Limit bounds the fan-out for
// delivery receipts; read receipts only need the first row.
func (s *messageRefStore) GetByUserUpTo(ctx context.Context, gateID, providerUserID string, watermark time.Time, limit int) ([]*sharedmodel.MessageRef, error) {
	const q = `
		SELECT gate_id, provider_message_id, provider_user_id, message_id, thread_id, member_id, domain_id, sent_at
		  FROM im_provider.message_refs
		 WHERE gate_id = $1
		   AND provider_user_id = $2
		   AND sent_at <= $3
		 ORDER BY sent_at DESC
		 LIMIT $4`

	rows, err := s.pool.Query(ctx, q, gateID, providerUserID, watermark, limit)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[sharedmodel.MessageRef])
}
