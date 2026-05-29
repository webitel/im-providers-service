package messaging

import (
	"time"

	"github.com/webitel/im-providers-service/internal/whatsapp/client"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type WhatsAppBusinessAccount struct {
	PhoneNumber   string `json:"phone_number" db:"phone_number"`
	PhoneNumberID string `json:"phone_number_id" db:"phone_number_id"`
	BusinessID    string `json:"business_id" db:"business_id"`

	AccessTokenEncrypted []byte     `json:"access_token_encrypted" db:"access_token_encrypted"`
	AccessTokenDecrypted string     `json:"-" db:"-"`
	AccessTokenExpiresAt *time.Time `json:"access_token_expires_at" db:"access_token_expires_at"`

	Contact common.Contact `json:"contact" db:"contact"`
}

func (whatsAppBusinessAccount *WhatsAppBusinessAccount) DeepClone() WhatsAppBusinessAccount {
	if whatsAppBusinessAccount == nil {
		return WhatsAppBusinessAccount{}
	}

	var tokenExpiresAt *time.Time
	if whatsAppBusinessAccount.AccessTokenExpiresAt != nil {
		expiresRef := *whatsAppBusinessAccount.AccessTokenExpiresAt
		tokenExpiresAt = &expiresRef
	}

	return WhatsAppBusinessAccount{
		PhoneNumber:          whatsAppBusinessAccount.PhoneNumber,
		PhoneNumberID:        whatsAppBusinessAccount.PhoneNumberID,
		BusinessID:           whatsAppBusinessAccount.BusinessID,
		AccessTokenEncrypted: whatsAppBusinessAccount.AccessTokenEncrypted,
		AccessTokenDecrypted: whatsAppBusinessAccount.AccessTokenDecrypted,
		AccessTokenExpiresAt: tokenExpiresAt,
		Contact: common.Contact{
			ID:  whatsAppBusinessAccount.Contact.GetID(),
			Iss: whatsAppBusinessAccount.Contact.GetIss(),
			Sub: whatsAppBusinessAccount.Contact.GetSub(),
		},
	}
}

func (whatsAppBusinessAccount *WhatsAppBusinessAccount) IsTokenExpired() bool {
	if whatsAppBusinessAccount == nil {
		return true
	}

	if whatsAppBusinessAccount.AccessTokenExpiresAt == nil {
		return false
	}

	isExpired := time.Now().UTC().After(
		whatsAppBusinessAccount.AccessTokenExpiresAt.UTC(),
	)

	return isExpired
}

func (whatsAppBusinessAccount *WhatsAppBusinessAccount) PostFetch(encyptor common.Encryptor) (WhatsAppBusinessAccount, error) {
	if whatsAppBusinessAccount == nil {
		return WhatsAppBusinessAccount{}, errors.InvalidArgument("receive nil pointer dereference call for WhatsAppBusinessAccount", errors.WithID("messaging.model.post_fetch"))
	}

	if whatsAppBusinessAccount.AccessTokenDecrypted != "" {
		return whatsAppBusinessAccount.DeepClone(), nil
	}

	decryptedAccessToken, err := encyptor.Decrypt(string(whatsAppBusinessAccount.AccessTokenEncrypted))
	if err != nil {
		return WhatsAppBusinessAccount{}, errors.Internal("decypting access token", errors.WithCause(err), errors.WithID("messaging.model.post_fetch"), errors.WithValue("phone_number_id", whatsAppBusinessAccount.PhoneNumberID))
	}

	clone := whatsAppBusinessAccount.DeepClone()
	clone.AccessTokenDecrypted = decryptedAccessToken

	return clone, nil
}

func (whatsAppBusinessAccount *WhatsAppBusinessAccount) CreateRequestClient() (*client.RequestClient, error) {
	return client.NewRequesClient(client.WithAccessTokenConfig(whatsAppBusinessAccount.AccessTokenDecrypted))
}
