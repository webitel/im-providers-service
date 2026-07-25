package viber

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupKeyTTL = 24 * time.Hour

// messageSeen atomically checks-and-marks a Viber message token as processed.
// Uses Redis SET NX EX so the operation is safe across multiple pod replicas.
//
// Returns true when the token was already seen (duplicate — skip processing).
// On Redis error it returns false to prefer at-least-once delivery over message loss.
func messageSeen(ctx context.Context, rdb *redis.Client, token string) bool {
	if token == "" {
		return false
	}
	inserted, err := rdb.SetNX(ctx, "viber:mtoken:"+token, 1, dedupKeyTTL).Result()
	if err != nil {
		return false
	}
	return !inserted
}
