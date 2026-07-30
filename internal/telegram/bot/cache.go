package bot

import (
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/google/uuid"
)

type idempotencyKey struct {
	gateID   uuid.UUID
	updateID int64
}

func newIdempotencyCache() (*defaultIdempotencyCache, error) {
	cache, err := lru.New[idempotencyKey, bool](200)
	if err != nil {
		return nil, err
	}
	return &defaultIdempotencyCache{
		lru: cache,
	}, nil
}

type defaultIdempotencyCache struct {
	lru *lru.Cache[idempotencyKey, bool]
}

func (c *defaultIdempotencyCache) IsProcessed(gateID uuid.UUID, updateID int64) bool {
	v, ok := c.lru.Get(idempotencyKey{gateID: gateID, updateID: updateID})
	return ok && v
}

func (c *defaultIdempotencyCache) MarkProcessed(gateID uuid.UUID, updateID int64) {
	c.lru.Add(idempotencyKey{gateID: gateID, updateID: updateID}, true)
}
