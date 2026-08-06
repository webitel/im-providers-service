package viber

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupKeyTTL = 24 * time.Hour

func messageSeen(ctx context.Context, rdb *redis.Client, token string) bool {
	if rdb == nil || token == "" {
		return false
	}
	inserted, err := rdb.SetNX(ctx, "viber:mtoken:"+token, 1, dedupKeyTTL).Result()
	if err != nil {
		return false
	}
	return !inserted
}
