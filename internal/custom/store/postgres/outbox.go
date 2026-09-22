package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
)

var _ customstore.OutboxStore = (*CustomStore)(nil)

type outboxRow struct {
	gateRow

	OutboxID  int64  `db:"outbox_id"`
	ChatKey   string `db:"chat_key"`
	MessageID string `db:"message_id"`
	Payload   []byte `db:"payload"`
	Attempt   int32  `db:"attempt"`
}

func (s *CustomStore) EnqueueOutbox(ctx context.Context, rec custommodel.OutboxRecord) error {
	const query = `
		INSERT INTO im_provider.gate_custom_outbox (gate_id, chat_key, message_id, payload, next_attempt_at)
		VALUES (@GateID, @ChatKey, @MessageID, @Payload, @NextAt)`

	if _, err := s.pool.Exec(ctx, query, outboxArgs(rec)); err != nil {
		return errors.Internal("failed to enqueue custom outbox record",
			errors.WithCause(err), errors.WithID("custom.store.postgres.enqueue_outbox"))
	}

	return nil
}

func (s *CustomStore) EnqueueOutboxIfPending(ctx context.Context, rec custommodel.OutboxRecord) (bool, error) {
	const query = `
		INSERT INTO im_provider.gate_custom_outbox (gate_id, chat_key, message_id, payload, next_attempt_at)
		SELECT @GateID, @ChatKey, @MessageID, @Payload, @NextAt
		WHERE EXISTS (
			SELECT 1 FROM im_provider.gate_custom_outbox
			WHERE gate_id = @GateID AND chat_key = @ChatKey AND status = 'pending'
		)`

	tag, err := s.pool.Exec(ctx, query, outboxArgs(rec))
	if err != nil {
		return false, errors.Internal("failed to enqueue custom outbox record behind pending ones",
			errors.WithCause(err), errors.WithID("custom.store.postgres.enqueue_outbox_if_pending"))
	}

	return tag.RowsAffected() > 0, nil
}

func (s *CustomStore) ClaimOutbox(ctx context.Context, limit int, lease time.Duration) ([]custommodel.OutboxTask, error) {
	const errID = "custom.store.postgres.claim_outbox"

	const query = `
	WITH due AS (
		SELECT DISTINCT ON (o.gate_id, o.chat_key) o.id
		FROM im_provider.gate_custom_outbox o
		WHERE o.status = 'pending'
		  AND o.next_attempt_at <= NOW()
		  AND NOT EXISTS (
			  SELECT 1 FROM im_provider.gate_custom_outbox busy
			  WHERE busy.gate_id = o.gate_id
			    AND busy.chat_key = o.chat_key
			    AND busy.locked_until > NOW()
		  )
		ORDER BY o.gate_id, o.chat_key, o.id
		LIMIT @Limit
	),
	claimed AS (
		SELECT o.id
		FROM im_provider.gate_custom_outbox o
		WHERE o.id IN (SELECT id FROM due)
		FOR UPDATE OF o SKIP LOCKED
	),
	leased AS (
		UPDATE im_provider.gate_custom_outbox o
		SET locked_until = NOW() + make_interval(secs => @LeaseSeconds)
		FROM claimed
		WHERE o.id = claimed.id
		RETURNING o.id, o.gate_id, o.chat_key, o.message_id, o.payload, o.attempt
	)
	SELECT
		l.id AS outbox_id,
		l.chat_key,
		l.message_id,
		l.payload,
		l.attempt,` + selectColumns + `
	FROM leased l
	JOIN im_provider.gates g ON g.id = l.gate_id
	JOIN im_provider.bots b ON b.gate_id = g.id
	JOIN im_provider.gate_custom c ON c.gate_id = g.id`

	rows, err := s.pool.Query(ctx, query, pgx.NamedArgs{
		"Limit":        limit,
		"LeaseSeconds": lease.Seconds(),
	})
	if err != nil {
		return nil, errors.Internal("failed to claim custom outbox records",
			errors.WithCause(err), errors.WithID(errID))
	}

	claimed, err := pgx.CollectRows(rows, pgx.RowToStructByName[outboxRow])
	if err != nil {
		return nil, errors.Internal("failed to scan custom outbox records",
			errors.WithCause(err), errors.WithID(errID))
	}

	tasks := make([]custommodel.OutboxTask, 0, len(claimed))

	for i := range claimed {
		gate := &custommodel.CustomGate{}
		if err := s.fillGate(&claimed[i].gateRow, gate); err != nil {
			return nil, err
		}

		tasks = append(tasks, custommodel.OutboxTask{
			ID:        claimed[i].OutboxID,
			ChatKey:   claimed[i].ChatKey,
			MessageID: claimed[i].MessageID,
			Payload:   claimed[i].Payload,
			Attempt:   claimed[i].Attempt,
			Gate:      gate,
		})
	}

	return tasks, nil
}

func (s *CustomStore) RetryOutbox(ctx context.Context, id int64, nextAt time.Time, lastErr string) error {
	const query = `
		UPDATE im_provider.gate_custom_outbox
		SET attempt = attempt + 1,
		    next_attempt_at = @NextAt,
		    last_error = @LastError,
		    locked_until = '-infinity'
		WHERE id = @ID`

	if _, err := s.pool.Exec(ctx, query, pgx.NamedArgs{
		"ID":        id,
		"NextAt":    nextAt,
		"LastError": lastErr,
	}); err != nil {
		return errors.Internal("failed to reschedule custom outbox record",
			errors.WithCause(err), errors.WithID("custom.store.postgres.retry_outbox"))
	}

	return nil
}

func (s *CustomStore) FailOutbox(ctx context.Context, id int64, lastErr string) error {
	const query = `
		UPDATE im_provider.gate_custom_outbox
		SET status = 'failed',
		    attempt = attempt + 1,
		    last_error = @LastError,
		    locked_until = '-infinity'
		WHERE id = @ID`

	if _, err := s.pool.Exec(ctx, query, pgx.NamedArgs{
		"ID":        id,
		"LastError": lastErr,
	}); err != nil {
		return errors.Internal("failed to mark custom outbox record failed",
			errors.WithCause(err), errors.WithID("custom.store.postgres.fail_outbox"))
	}

	return nil
}

func (s *CustomStore) DeleteOutbox(ctx context.Context, id int64) error {
	const query = `DELETE FROM im_provider.gate_custom_outbox WHERE id = @ID`

	if _, err := s.pool.Exec(ctx, query, pgx.NamedArgs{"ID": id}); err != nil {
		return errors.Internal("failed to delete custom outbox record",
			errors.WithCause(err), errors.WithID("custom.store.postgres.delete_outbox"))
	}

	return nil
}

func outboxArgs(rec custommodel.OutboxRecord) pgx.NamedArgs {
	nextAt := rec.NextAt
	if nextAt.IsZero() {
		nextAt = time.Now()
	}

	return pgx.NamedArgs{
		"GateID":    rec.GateID,
		"ChatKey":   rec.ChatKey,
		"MessageID": rec.MessageID,
		"Payload":   rec.Payload,
		"NextAt":    nextAt,
	}
}
