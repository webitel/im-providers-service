package model

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

//go:generate stringer -type=GateType,GateStatus -linecomment

type (
	GateType   int
	GateStatus int
)

const (
	TypeUnknown     GateType = iota // unknown
	TypeFacebook                    // facebook
	TypeInstagram                   // instagram
	TypeWhatsApp                    // whatsapp
	TypeTelegramBot                 // telegram_bot
	TypeTelegramApp                 // telegram_app
)

const (
	StatusUnknown  GateStatus = iota // unknown
	StatusActive                     // active
	StatusDisabled                   // disabled
	StatusError                      // error
)

// --- Universal Scanner for GateType ---

func (gt *GateType) Scan(value interface{}) error {
	if value == nil {
		*gt = TypeUnknown
		return nil
	}

	s, err := asString(value)
	if err != nil {
		return fmt.Errorf("scan GateType: %w", err)
	}

	*gt = ParseGateType(s)
	return nil
}

func (gt GateType) Value() (driver.Value, error) {
	// Returns the string representation (e.g., "facebook") to the DB
	return gt.String(), nil
}

// --- Universal Scanner for GateStatus ---

func (gs *GateStatus) Scan(value interface{}) error {
	if value == nil {
		*gs = StatusUnknown
		return nil
	}

	s, err := asString(value)
	if err != nil {
		return fmt.Errorf("scan GateStatus: %w", err)
	}

	*gs = ParseGateStatus(s)
	return nil
}

func (gs GateStatus) Value() (driver.Value, error) {
	return gs.String(), nil
}

// --- Helpers & Parsers ---

// asString safely converts database values (string or []byte) to string.
func asString(src any) (string, error) {
	switch v := src.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	}
	return "", fmt.Errorf("unexpected type: %T", src)
}

func ParseGateType(s string) GateType {
	val := strings.ToLower(strings.TrimSpace(s))
	// Internal mapping
	m := map[string]GateType{
		"facebook":     TypeFacebook,
		"instagram":    TypeInstagram,
		"whatsapp":     TypeWhatsApp,
		"telegram_bot": TypeTelegramBot,
		"telegram_app": TypeTelegramApp,
	}
	if v, ok := m[val]; ok {
		return v
	}
	return TypeUnknown
}

func ParseGateStatus(s string) GateStatus {
	val := strings.ToLower(strings.TrimSpace(s))
	m := map[string]GateStatus{
		"active":   StatusActive,
		"disabled": StatusDisabled,
		"error":    StatusError,
	}
	if v, ok := m[val]; ok {
		return v
	}
	return StatusUnknown
}
