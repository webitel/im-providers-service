package instagram

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupKeyTTL = 24 * time.Hour

// messageSeen atomically checks-and-marks an Instagram message ID as processed.
// Uses Redis SET NX EX so the operation is safe across multiple pod replicas.
//
// Returns true when the mid was already seen (duplicate — skip processing).
// Returns false when the mid is new (first time — process normally).
// On Redis error, returns false to prefer at-least-once delivery over message loss.
func messageSeen(ctx context.Context, rdb *redis.Client, mid string) bool {
	if mid == "" {
		return false
	}
	// SetNX returns true when the key was newly inserted.
	inserted, err := rdb.SetNX(ctx, "ig:mid:"+mid, 1, dedupKeyTTL).Result()
	if err != nil {
		return false
	}

	return !inserted
}
