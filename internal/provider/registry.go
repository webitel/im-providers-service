package provider

import (
	"fmt"
	"sync"
)

// Registry is a thread-safe map of registered provider adapters keyed by Type().
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry builds a Registry from a slice of providers (populated via fx value group).
func NewRegistry(providers []Provider) *Registry {
	reg := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, p := range providers {
		reg.providers[p.Type()] = p
	}
	return reg
}

// Get retrieves a provider by its type identifier. Returns an error if not registered.
func (r *Registry) Get(pType string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[pType]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", pType)
	}
	return p, nil
}
