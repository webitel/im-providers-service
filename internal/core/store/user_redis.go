package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

var _ ExternalUserCache = (*redisUserCache)(nil)

type redisUserCache struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewRedisUserCache initializes the Redis-based identity cache
func NewRedisUserCache(rdb *redis.Client, ttl time.Duration) ExternalUserCache {
	return &redisUserCache{
		rdb: rdb,
		ttl: ttl,
	}
}

func (r *redisUserCache) IsKnown(ctx context.Context, user *sharedmodel.ExternalUser) (bool, error) {
	// Key format: usr:hash:<sha256_of_id_and_names>
	key := "usr:hash:" + user.Hash()
	exists, err := r.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *redisUserCache) MarkKnown(ctx context.Context, user *sharedmodel.ExternalUser) error {
	key := "usr:hash:" + user.Hash()
	// Set with TTL to allow periodic re-syncing/verification
	return r.rdb.Set(ctx, key, "1", r.ttl).Err()
}
