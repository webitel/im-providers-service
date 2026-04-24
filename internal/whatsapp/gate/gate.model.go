package gate

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type WhatsAppBusinessAccountGate struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	MetaAppID            uuid.UUID `json:"meta_app_id" db:"meta_app_id"`
	PhoneNumber          string    `json:"phone_number" db:"phone_number"`
	PhoneNumberID        string    `json:"phone_number_id" db:"phone_number_id"`
	AccessToken          string    `json:"-" db:"-"`
	AccessTokenEncrypted []byte    `json:"access_token" db:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at" db:"access_token_expires_at"`
	BusinessID           string    `json:"business_id" db:"business_id"`
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
