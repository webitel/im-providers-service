package gate

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

const WhatsAppGateType string = "whatsapp"

type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type Peer interface {
	GetID() uuid.UUID
	GetSub() string
	GetIss() string
}

type InternalContact struct {
	ID  uuid.UUID `json:"id" db:"id"`
	Sub string    `json:"sub" db:"sub"`
	Iss string    `json:"iss" db:"iss"`
}

func (internalContact *InternalContact) GetID() uuid.UUID { return internalContact.ID }
func (internalContact *InternalContact) GetSub() string   { return internalContact.Sub }
func (internalContact *InternalContact) GetIss() string   { return internalContact.Iss }

type Gate struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	WhatsAppBusinessAccountGate WhatsAppBusinessAccountGate `json:"whats_app_business_account_gate" db:"whats_app_business_account_gate"`
}

func (gate *Gate) Validate() error {
	if gate == nil {
		return errors.InvalidArgument("gate is required", errors.WithID("gate.model.validate"))
	}

	if gate.Name == "" || strings.Trim(gate.Name, " ") == "" {
		return errors.InvalidArgument("gate name is required", errors.WithID("gate.model.validate"))
	}

	if gate.Type != WhatsAppGateType {
		return errors.InvalidArgument("invalid gate type, expecting 'whatsapp'", errors.WithID("gate.model.validate"), errors.WithID("gate.model.validate"))
	}

	if err := gate.WhatsAppBusinessAccountGate.Validate(); err != nil {
		return err
	}

	return nil
}

type WhatsAppBusinessAccountGate struct {
	ID                   uuid.UUID        `json:"id" db:"id"`
	MetaAppID            uuid.UUID        `json:"meta_app_id" db:"meta_app_id"`
	PhoneNumber          string           `json:"phone_number" db:"phone_number"`
	PhoneNumberID        string           `json:"phone_number_id" db:"phone_number_id"`
	AccessToken          string           `json:"-" db:"-"`
	AccessTokenEncrypted []byte           `json:"access_token" db:"access_token"`
	AccessTokenExpiresAt *time.Time       `json:"access_token_expires_at" db:"access_token_expires_at"`
	BusinessID           string           `json:"business_id" db:"business_id"`
	ContactID            uuid.UUID        `json:"-" db:"contact_id"`
	Contact              *InternalContact `json:"contact" db:"contact"`
}

func (gate *WhatsAppBusinessAccountGate) Validate() error {
	if gate == nil {
		return errors.InvalidArgument("gate can`t be null", errors.WithID("gate.model.validate"))
	}

	if gate.PhoneNumberID == "" || strings.Trim(gate.PhoneNumber, " ") == "" {
		return errors.InvalidArgument("WhatsApp phone number is required", errors.WithID("gate.model.validate"))
	}

	if gate.PhoneNumberID == "" || strings.Trim(gate.PhoneNumberID, " ") == "" {
		return errors.InvalidArgument("WhatsApp phone number id is required", errors.WithID("gate.model.validate"))
	}

	if gate.BusinessID == "" || strings.Trim(gate.BusinessID, " ") == "" {
		return errors.InvalidArgument("WhatsApp business id is required", errors.WithID("gate.model.validate"))
	}

	if gate.AccessToken == "" || strings.Trim(gate.AccessToken, " ") == "" {
		return errors.InvalidArgument("WhatsApp access token is required", errors.WithID("gate.model.validate"))
	}

	if gate.ContactID == uuid.Nil {
		return errors.InvalidArgument("gate contact binding is required", errors.WithID("gate.model.validate"))
	}

	return nil
}

func (gate *WhatsAppBusinessAccountGate) PreSave(encryptor Encryptor) (WhatsAppBusinessAccountGate, error) {
	encryptedToken, err := encryptor.Encrypt(gate.AccessToken)
	if err != nil {
		return WhatsAppBusinessAccountGate{}, errors.Internal("encrypting WABA access token", errors.WithID("gate.model.pre_save"), errors.WithCause(err))
	}

	prepared := WhatsAppBusinessAccountGate{
		MetaAppID:            gate.MetaAppID,
		PhoneNumber:          gate.PhoneNumber,
		PhoneNumberID:        gate.PhoneNumberID,
		AccessToken:          gate.AccessToken,
		AccessTokenEncrypted: []byte(encryptedToken),
		AccessTokenExpiresAt: gate.AccessTokenExpiresAt,
		BusinessID:           gate.BusinessID,
	}

	return prepared, nil
}

func (gate *WhatsAppBusinessAccountGate) PostFetch(encryptor Encryptor) (WhatsAppBusinessAccountGate, error) {
	decryptedToken, err := encryptor.Decrypt(string(gate.AccessTokenEncrypted))
	if err != nil {
		return WhatsAppBusinessAccountGate{}, errors.Internal("decrypting WABA access token", errors.WithCause(err), errors.WithID("gate.model.post_fetch"))
	}

	prepared := WhatsAppBusinessAccountGate{
		ID:                   gate.ID,
		MetaAppID:            gate.MetaAppID,
		PhoneNumber:          gate.PhoneNumber,
		PhoneNumberID:        gate.PhoneNumberID,
		AccessToken:          decryptedToken,
		AccessTokenExpiresAt: gate.AccessTokenExpiresAt,
		BusinessID:           gate.BusinessID,
	}

	return prepared, nil
}
