package model

import (
	"github.com/google/uuid"
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
)

type Gate struct {
	ID        uuid.UUID
	Name      string
	Bot       *coremodel.Peer
	Status    coremodel.GateStatus
	CreatedAt int64
	UpdatedAt int64
}

type CreateGate struct {
	Name   string
	Token  string
	Bot    *coremodel.Peer
	Status coremodel.GateStatus
}

type UpdateGate struct {
	ID     uuid.UUID
	Name   *string
	Token  *string
	Bot    *coremodel.Peer
	Status coremodel.GateStatus
}
