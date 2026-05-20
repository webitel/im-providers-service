package store

import (
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
)

var _ GateCache = (*lruCache)(nil)

type lruCache struct {
	// gates is a universal storage for all provider types (FB, WA, TG, etc.)
	gates *lru.Cache[string, GateState]
}

// NewLRUCache creates a new universal LRU-based cache with a fixed size.
// Size determines how many unique gates can be kept in memory.
func NewLRUCache(size int) (GateCache, error) {
	c, err := lru.New[string, GateState](size)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize lru cache: %w", err)
	}
	return &lruCache{gates: c}, nil
}

// Set adds or updates a gate in the cache.
func (c *lruCache) Set(key string, state GateState) {
	c.gates.Add(key, state)
}

// Get attempts to find a gate by its unique provider key.
func (c *lruCache) Get(key string) (GateState, bool) {
	return c.gates.Get(key)
}

// Delete invalidates a specific gate's cache entry.
func (c *lruCache) Delete(key string) {
	c.gates.Remove(key)
}
