// Package model defines the core domain entities and business logic rules
// for the IM Thread service. This package is the heart of the application
// and must remain independent of any external frameworks or transport layers.
package model

type PeerType int16

//go:generate stringer -type=PeerType
const (
	PeerContact PeerType = iota + 1
	PeerGroup
	PeerChannel
)

type Peer struct {
	ID     string
	Issuer string
	Type   PeerType
}
