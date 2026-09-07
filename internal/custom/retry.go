package custom

import (
	"context"
	"errors"
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"go.uber.org/fx"

	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

const (
	retryShards     = 8
	retryQueueDepth = 256
	retryBaseDelay  = time.Second
	retryMaxDelay   = 30 * time.Second
)

// retryQueue re-delivers payloads the external system did not accept on the
// first attempt. Work is sharded by conversation so that messages of one chat
// keep their order and one unreachable customer cannot stall the others; the
// first attempt always happens inline in the caller, so a healthy channel never
// touches this queue at all.
type retryQueue struct {
	logger *slog.Logger
	api    *client
	status *sharedsvc.StatusReporter

	shards []chan retryTask
	wg     sync.WaitGroup
	cancel context.CancelFunc
	once   sync.Once
}

type retryTask struct {
	gate      *custommodel.CustomGate
	chatKey   string
	payload   []byte
	messageID string
	attempt   int32
	lastErr   error
}

func newRetryQueue(logger *slog.Logger, api *client, status *sharedsvc.StatusReporter, lc fx.Lifecycle) *retryQueue {
	q := &retryQueue{
		logger: logger.With("component", "custom.retry"),
		api:    api,
		status: status,
		shards: make([]chan retryTask, retryShards),
	}

	for i := range q.shards {
		q.shards[i] = make(chan retryTask, retryQueueDepth)
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

	for i := range q.shards {
		q.wg.Add(1)

		go func(in <-chan retryTask) {
			defer q.wg.Done()

			q.run(ctx, in)
		}(q.shards[i])
	}
}

func (q *retryQueue) stop() {
	q.once.Do(func() {
		if q.cancel != nil {
			q.cancel()
		}

		q.wg.Wait()
	})
}

// enqueue schedules another attempt. A full shard means the channel is already
// far behind, so the message is failed now rather than queued behind a backlog
// nobody is draining.
func (q *retryQueue) enqueue(ctx context.Context, task retryTask) {
	if task.attempt >= task.gate.RetryAttempts {
		q.exhausted(ctx, task)

		return
	}

	shard := q.shards[shardOf(task.chatKey)]

	select {
	case shard <- task:
	default:
		q.logger.WarnContext(ctx, "retry queue full, failing message now",
			"gate_id", task.gate.ID,
			"message_id", task.messageID,
		)
		q.exhausted(ctx, task)
	}
}

func (q *retryQueue) run(ctx context.Context, in <-chan retryTask) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-in:
			q.attempt(ctx, task)
		}
	}
}

func (q *retryQueue) attempt(ctx context.Context, task retryTask) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(backoff(task.attempt)):
	}

	err := q.api.post(ctx, task.gate, task.payload)
	if err == nil {
		q.logger.InfoContext(ctx, "callback delivered on retry",
			"gate_id", task.gate.ID,
			"message_id", task.messageID,
			"attempt", task.attempt+1,
		)

		return
	}

	task.attempt++
	task.lastErr = err

	if errors.Is(err, custommodel.ErrCallbackRejected) {
		q.exhausted(ctx, task)

		return
	}

	q.enqueue(ctx, task)
}

// exhausted turns a permanently undelivered message into a visible failure, so
// the operator sees the message did not arrive rather than believing it did.
func (q *retryQueue) exhausted(ctx context.Context, task retryTask) {
	q.logger.ErrorContext(ctx, "callback delivery gave up",
		"gate_id", task.gate.ID,
		"message_id", task.messageID,
		"attempts", task.attempt,
		"err", task.lastErr,
	)

	if q.status == nil || task.messageID == "" {
		return
	}

	message := ""
	if task.lastErr != nil {
		message = task.lastErr.Error()
	}

	q.status.FailedByProviderID(ctx, task.gate.ID, task.messageID, time.Now(), "callback_undelivered", message)
}

func backoff(attempt int32) time.Duration {
	delay := retryBaseDelay << attempt
	if delay > retryMaxDelay || delay <= 0 {
		delay = retryMaxDelay
	}

	return delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
}

func shardOf(key string) int {
	if key == "" {
		return 0
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(key))

	return int(h.Sum32() % retryShards)
}
