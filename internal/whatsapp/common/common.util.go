package common

import "github.com/google/uuid"

func SafeConvertStringToUUID(idStr string) uuid.UUID {
	id, _ := uuid.Parse(idStr)
	return id
}
