// Package via stamps the gate an inbound message arrived through onto calls
// forwarded to im-gateway-service. The gate id disambiguates the same external
// contact sub across gates; name and type let im-thread-service show and sort
// threads by their source without calling back into this service.
package via

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const (
	HeaderID   = "x-webitel-via"
	HeaderType = "x-webitel-via-type"
	// Gate names are user-defined and may be non-ASCII, which plain gRPC
	// metadata values cannot carry; grpc-go base64-encodes "-bin" values.
	HeaderName = "x-webitel-via-name-bin"
)

// Gate identifies the gate a forwarded call belongs to.
type Gate struct {
	ID   string
	Type string
	Name string
}

// Pairs returns the metadata key/value pairs for g. Empty type and name are
// omitted, so the gateway falls back to the gate id alone.
func (g Gate) Pairs() []string {
	kv := []string{HeaderID, g.ID}
	if g.Type != "" {
		kv = append(kv, HeaderType, g.Type)
	}

	if g.Name != "" {
		kv = append(kv, HeaderName, g.Name)
	}

	return kv
}

// AppendToOutgoingContext adds g's headers to the outgoing metadata of ctx.
func AppendToOutgoingContext(ctx context.Context, g Gate) context.Context {
	return metadata.AppendToOutgoingContext(ctx, g.Pairs()...)
}
