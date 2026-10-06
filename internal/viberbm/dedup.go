package viberbm

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupKeyTTL = 24 * time.Hour

// messageDedup marks InfoBip messageIds so redeliveries are skipped, and
// releases a mark when the forward to core failed so the redelivery is kept.
type messageDedup interface {
	Seen(ctx context.Context, mid string) bool
	Forget(ctx context.Context, mid string)
}

type redisDedup struct {
	rdb *redis.Client
}

// Seen atomically checks-and-marks via SET NX EX (safe across replicas).
// Fails open on a Redis error, preferring at-least-once delivery over loss.
func (d redisDedup) Seen(ctx context.Context, mid string) bool {
	if mid == "" {
		return false
	}

	inserted, err := d.rdb.SetNX(ctx, dedupKey(mid), 1, dedupKeyTTL).Result()
	if err != nil {
		return false
	}

	return !inserted
}

func (d redisDedup) Forget(ctx context.Context, mid string) {
	if mid == "" {
		return
	}

	_ = d.rdb.Del(ctx, dedupKey(mid)).Err()
}

func dedupKey(mid string) string { return "viberbm:mid:" + mid }
