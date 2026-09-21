package store

import (
	"context"

	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

var _ CustomStore = (*CachedStore)(nil)

type CachedStore struct {
	CustomStore

	cache sharedstore.GateCache
}

func NewCachedStore(next CustomStore, cache sharedstore.GateCache) *CachedStore {
	return &CachedStore{CustomStore: next, cache: cache}
}

func (s *CachedStore) Insert(ctx context.Context, dc int64, g *custommodel.CustomGate) error {
	if err := s.CustomStore.Insert(ctx, dc, g); err != nil {
		return err
	}

	s.invalidate(g.WebhookURI)

	return nil
}

func (s *CachedStore) Update(ctx context.Context, req custommodel.UpdateCustom) (*custommodel.CustomGate, error) {
	gate, err := s.CustomStore.Update(ctx, req)
	if err != nil {
		return nil, err
	}

	s.invalidate(gate.WebhookURI)

	return gate, nil
}

func (s *CachedStore) Unbind(ctx context.Context, gateID string) (*custommodel.CustomGate, error) {
	gate, err := s.CustomStore.Unbind(ctx, gateID)
	if err != nil {
		return nil, err
	}

	s.invalidate(gate.WebhookURI)

	return gate, nil
}

func (s *CachedStore) invalidate(webhookURI string) {
	if webhookURI == "" {
		return
	}

	s.cache.Delete(webhookURI)
}
