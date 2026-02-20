package model

import (
	"strings"

	"github.com/google/uuid"
)

//go:generate stringer -type=PeerType
type PeerType int16

const (
	PeerUser PeerType = iota + 1
	PeerGroup
	PeerChannel
)

type Peer struct {
	ID     uuid.UUID `json:"id"`
	Type   PeerType  `json:"type"`
	Sub    string    `json:"sub,omitempty"`
	Issuer string    `json:"issuer,omitempty"`
	Name   string    `json:"name,omitempty"`
}

type PeerOption func(*Peer)

// WithIdentity applies enrichment data from external services
func WithIdentity(sub, issuer, name string) PeerOption {
	return func(p *Peer) {
		p.Sub = sub
		p.Issuer = issuer
		p.Name = name
	}
}

func NewPeer(id uuid.UUID, pType PeerType, opts ...PeerOption) Peer {
	p := Peer{ID: id, Type: pType}
	for _, opt := range opts {
		opt(&p)
	}
	return p
}

// IsEnriched determines if the peer has verified identity metadata
func (p Peer) IsEnriched() bool {
	return p.Sub != ""
}

// GetRoutingParts returns normalized segments for RabbitMQ routing keys
func (p Peer) GetRoutingParts() (sub, issuer string) {
	sub, issuer = "any", "any"
	if p.Sub != "" {
		sub = strings.ToLower(p.Sub)
	}
	if p.Issuer != "" {
		issuer = strings.ToLower(p.Issuer)
	}
	return
}
