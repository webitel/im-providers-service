package client

import "context"

// IdentityProvider defines an object that can provide its identification string.
type IdentityProvider interface {
	Identity() string
}

// StringIdentity is a convenience type that lets any string satisfy IdentityProvider.
type StringIdentity string

func (s StringIdentity) Identity() string { return string(s) }

type authKey struct{}

// WithIdentity injects an IdentityProvider into the context.
// This allows the interceptor to retrieve the ID later.
func WithIdentity(ctx context.Context, p IdentityProvider) context.Context {
	return context.WithValue(ctx, authKey{}, p)
}

// GetIdentity retrieves the IdentityProvider from context.
func GetIdentity(ctx context.Context) (IdentityProvider, bool) {
	val, ok := ctx.Value(authKey{}).(IdentityProvider)
	return val, ok
}
