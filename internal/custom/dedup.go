package custom

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const dedupKeyTTL = 24 * time.Hour

func messageSeen(ctx context.Context, rdb *redis.Client, gateID, externalID string) bool {
	if rdb == nil || externalID == "" {
		return false
	}

	inserted, err := rdb.SetNX(ctx, "custom:"+gateID+":"+externalID, 1, dedupKeyTTL).Result()
	if err != nil {
		return false
	}

	return !inserted
}
