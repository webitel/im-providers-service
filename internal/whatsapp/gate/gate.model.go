package gate

import (
	"time"

	"github.com/google/uuid"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type WabaGate struct {
	ID                   uuid.UUID `json:"id" db:"id"`
	MetaAppID            uuid.UUID `json:"meta_app_id" db:"meta_app_id"`
	PhoneNumber          string    `json:"phone_number" db:"phone_number"`
	PhoneNumberID        string    `json:"phone_number_id" db:"phone_number_id"`
	AccessToken          string    `json:"-" db:"-"`
	AccessTokenEncrypted []byte    `json:"-" db:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at" db:"access_token_expires_at"`
	BusinessID           string    `json:"business_id" db:"business_id"`
}

func (gate *WabaGate) PreSave(encryptor Encryptor) (WabaGate, error) {
	encryptedToken, err := encryptor.Encrypt(gate.AccessToken)
	if err != nil {
		return WabaGate{}, errors.Internal("encrypting WABA access token", errors.WithID("gate.model.pre_save"), errors.WithCause(err))
	}

	prepared := WabaGate{
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

func (gate *WabaGate) PostFetch(encryptor Encryptor) (WabaGate, error) {
	decryptedToken, err := encryptor.Decrypt(string(gate.AccessTokenEncrypted))
	if err != nil {
		return WabaGate{}, errors.Internal("decrypting WABA access token", errors.WithCause(err), errors.WithID("gate.model.post_fetch"))
	}

	prepared := WabaGate{
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
