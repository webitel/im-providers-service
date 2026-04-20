package provider

import (
	"fmt"
	"sync"
)

// Registry manages all registered messaging providers.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new registry instance.
func NewRegistry(providers []Provider) *Registry {
	reg := &Registry{
		providers: make(map[string]Provider),
	}

	for _, p := range providers {
		reg.providers[p.Type()] = p
	}

	return reg
}

// Get retrieves a provider by its unique type identifier (e.g., "facebook").
func (r *Registry) Get(pType string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[pType]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", pType)
	}
	return p, nil
}
