package store

import (
	"context"
	"fmt"

	"github.com/webitel/webitel-go-kit/pkg/cache"
)

var _ GateCache = (*gateCache)(nil)

type gateCache struct {
	inner cache.Cache[string, GateState]
}

// NewLRUCache creates a universal in-memory gate cache backed by Ristretto (W-TinyLFU).
// Size determines the maximum number of unique gate entries kept in memory.
func NewLRUCache(size int) (GateCache, error) {
	c, err := cache.New[string, GateState]().
		L1(cache.RistrettoConfig{
			MaxCost:     int64(size),
			NumCounters: int64(size) * 10,
		}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gate cache: %w", err)
	}
	return &gateCache{inner: c}, nil
}

func (c *gateCache) Set(key string, state GateState) {
	_ = c.inner.Set(context.Background(), key, state)
}

func (c *gateCache) Get(key string) (GateState, bool) {
	v, ok, _ := c.inner.Get(context.Background(), key)
	return v, ok
}

func (c *gateCache) Delete(key string) {
	_ = c.inner.Delete(context.Background(), key)
}
