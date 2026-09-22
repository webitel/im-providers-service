package custom

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
)

const (
	retryBatchSize  = 16
	retryPollPeriod = time.Second
	retryLease      = 2 * time.Minute
	retryBaseDelay  = time.Second
	retryMaxDelay   = 30 * time.Second
	outboxWriteWait = 5 * time.Second
)

type retryQueue struct {
	logger *slog.Logger
	api    *client
	status *sharedsvc.StatusReporter
	outbox customstore.OutboxStore

	cancel context.CancelFunc
	wg     sync.WaitGroup
	once   sync.Once
}

func newRetryQueue(
	logger *slog.Logger,
	api *client,
	status *sharedsvc.StatusReporter,
	outbox customstore.OutboxStore,
	lc fx.Lifecycle,
) *retryQueue {
	q := &retryQueue{
		logger: logger.With("component", "custom.retry"),
		api:    api,
		status: status,
		outbox: outbox,
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			q.start()

			return nil
		},
		OnStop: func(context.Context) error {
			q.stop()

			return nil
		},
	})

	return q
}

func (q *retryQueue) start() {
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel

	q.wg.Add(1)

	go func() {
		defer q.wg.Done()

		q.poll(ctx)
	}()
}

func (q *retryQueue) stop() {
	q.once.Do(func() {
		if q.cancel != nil {
			q.cancel()
		}

		q.wg.Wait()
	})
}

func (q *retryQueue) poll(ctx context.Context) {
	ticker := time.NewTicker(retryPollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.drain(ctx)
		}
	}
}

func (q *retryQueue) drain(ctx context.Context) {
	tasks, err := q.outbox.ClaimOutbox(ctx, retryBatchSize, retryLease)
	if err != nil {
		if ctx.Err() == nil {
			q.logger.ErrorContext(ctx, "failed to claim outbox records", "err", err)
		}

		return
	}

	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)

		go func(task custommodel.OutboxTask) {
			defer wg.Done()

			q.attempt(ctx, task)
		}(task)
	}

	wg.Wait()
}

func (q *retryQueue) enqueue(ctx context.Context, rec custommodel.OutboxRecord) error {
	rec.NextAt = time.Now().Add(backoff(0))

	return q.outbox.EnqueueOutbox(ctx, rec)
}

func (q *retryQueue) enqueueBehindPending(ctx context.Context, rec custommodel.OutboxRecord) (bool, error) {
	rec.NextAt = time.Now()

	return q.outbox.EnqueueOutboxIfPending(ctx, rec)
}

func (q *retryQueue) attempt(ctx context.Context, task custommodel.OutboxTask) {
	err := q.api.post(ctx, task.Gate, task.Payload)

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), outboxWriteWait)
	defer cancel()

	if err == nil {
		q.logger.InfoContext(ctx, "callback delivered on retry",
			"gate_id", task.Gate.ID,
			"message_id", task.MessageID,
			"attempt", task.Attempt+1,
		)

		if delErr := q.outbox.DeleteOutbox(writeCtx, task.ID); delErr != nil {
			q.logger.ErrorContext(ctx, "delivered payload left in the outbox, it will be sent again",
				"message_id", task.MessageID,
				"err", delErr,
			)
		}

		return
	}

	attempt := task.Attempt + 1

	if errors.Is(err, custommodel.ErrCallbackRejected) || attempt >= task.Gate.RetryAttempts {
		q.exhausted(writeCtx, task, attempt, err)

		return
	}

	if rErr := q.outbox.RetryOutbox(writeCtx, task.ID, time.Now().Add(backoff(attempt)), err.Error()); rErr != nil {
		q.logger.ErrorContext(ctx, "failed to reschedule outbox record",
			"message_id", task.MessageID,
			"err", rErr,
		)
	}
}

// exhausted turns a permanently undelivered message into a visible failure, so
// the operator sees the message did not arrive rather than believing it did.
func (q *retryQueue) exhausted(ctx context.Context, task custommodel.OutboxTask, attempt int32, cause error) {
	q.logger.ErrorContext(ctx, "callback delivery gave up",
		"gate_id", task.Gate.ID,
		"message_id", task.MessageID,
		"attempts", attempt,
		"err", cause,
	)

	message := ""
	if cause != nil {
		message = cause.Error()
	}

	if err := q.outbox.FailOutbox(ctx, task.ID, message); err != nil {
		q.logger.ErrorContext(ctx, "failed to mark outbox record failed",
			"message_id", task.MessageID,
			"err", err,
		)
	}

	if q.status == nil || task.MessageID == "" {
		return
	}

	q.status.FailedByProviderID(ctx, task.Gate.ID, task.MessageID, time.Now(), "callback_undelivered", message)
}

func backoff(attempt int32) time.Duration {
	delay := retryBaseDelay << attempt
	if delay > retryMaxDelay || delay <= 0 {
		delay = retryMaxDelay
	}

	return delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
}
