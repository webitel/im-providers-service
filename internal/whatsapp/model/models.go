package model

import (
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// WhatsAppGate represents a WhatsApp Business gate configuration.
type WhatsAppGate struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	MetaAppID     string                 `json:"meta_app_id"`
	WABAID        string                 `json:"waba_id"`
	PhoneNumberID string                 `json:"phone_number_id"`
	PhoneDisplay  string                 `json:"phone_display"`
	AccessToken   string                 `json:"-"`
	Status        sharedmodel.GateStatus `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// CreateWhatsApp is the request payload for creating a new WhatsApp Business gate.
type CreateWhatsApp struct {
	Name          string
	MetaAppID     string
	WABAID        string
	PhoneNumberID string
	AccessToken   string
}

// UpdateWhatsApp is the request payload for updating an existing WhatsApp Business gate.
type UpdateWhatsApp struct {
	ID          string
	Name        *string
	AccessToken *string
}
