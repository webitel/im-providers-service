package model

import "github.com/google/uuid"

type GetGateFilters struct {
	ID  *uuid.UUID
	DC  *int64
	URI *string
}
