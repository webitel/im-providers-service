package auth

import (
	"context"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var IdentityNotFoundErr = errors.New("identity not found in the context")

type contextKey string

const (
	// AuthContextKey is used to store/retrieve Identity from context
	AuthContextKey contextKey = "auth_identity"

	SchemaIdentificationHeader = "x-webitel-schema"
	XWebitelTypeHeader         = "x-webitel-type"
)

type XWebitelType string

const (
	XWebitelTypeSchema XWebitelType = "schema"
	XWebitelTypeEngine XWebitelType = "engine"
)

type Authorizer interface {
	SetIdentity(ctx context.Context) (context.Context, error)
}

// Identity stores the resolved domain ownership and contact ID
type Identifier interface {
	GetContactID() string
	GetDomainID() int64
	GetName() string
}

// GetIdentity is a helper to extract the identity from context safely.
func GetIdentityFromContext(ctx context.Context) (Identifier, bool) {
	id, ok := ctx.Value(AuthContextKey).(Identifier)

	return id, ok
}
