package util

import (
	"time"

	"github.com/google/uuid"
)

func SafeParseUUID(s string) uuid.UUID {
	val, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return val
}

func SafeParseRFC3339(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now().UnixMilli()
	}
	return t.UnixMilli()
}
