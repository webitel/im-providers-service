package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/webitel-go-kit/pkg/cache"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

var _ ExternalUserCache = (*redisUserCache)(nil)

type redisUserCache struct {
	identity cache.Cache[string, string] // usr:hash:{sha256} → "1"
}

// NewRedisUserCache initializes the Redis-backed external user identity cache.
// RawString codec preserves the existing wire format ("1") stored in Redis.
func NewRedisUserCache(rdb *redis.Client, ttl time.Duration) (ExternalUserCache, error) {
	identity, err := cache.New[string, string]().
		L2(cache.RedisConfig[string]{
			Client: rdb,
			Prefix: "usr:hash",
			TTL:    ttl,
			Codec:  cache.RawString(),
		}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user identity cache: %w", err)
	}

	return &redisUserCache{identity: identity}, nil
}

func (r *redisUserCache) IsKnown(ctx context.Context, user *sharedmodel.ExternalUser) (bool, error) {
	_, ok, err := r.identity.Get(ctx, user.Hash())
	return ok, err
}

func (r *redisUserCache) MarkKnown(ctx context.Context, user *sharedmodel.ExternalUser) error {
	return r.identity.Set(ctx, user.Hash(), "1")
}

