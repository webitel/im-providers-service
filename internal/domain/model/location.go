package model

import (
	"github.com/google/uuid"
)

type SendResponse struct {
	ID uuid.UUID
	To Peer
}

type SendLocationRequest struct {
	From       Peer
	To         Peer
	Latitude   float64
	Longitude  float64
	Name       *string
	Address    *string
	ExternalID string
	DomainID   int
}

type SendContactRequest struct {
	From        Peer
	To          Peer
	Name        *string
	Email       *string
	PhoneNumber *string
	Metadata    map[string]any
	ExternalID  string
	DomainID    int
}
