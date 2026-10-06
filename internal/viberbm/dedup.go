package viberbm

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupKeyTTL = 24 * time.Hour

// messageSeen atomically checks-and-marks an InfoBip messageId via SET NX EX
// (safe across replicas). Returns true if already seen. Fails open on a Redis
// error, preferring at-least-once delivery over silent loss.
func messageSeen(ctx context.Context, rdb *redis.Client, mid string) bool {
	if mid == "" {
		return false
	}

	inserted, err := rdb.SetNX(ctx, "viberbm:mid:"+mid, 1, dedupKeyTTL).Result()
	if err != nil {
		return false
	}

	return !inserted
}
