package common

import (
	"time"

	"github.com/google/uuid"
	"github.com/webitel/im-providers-service/internal/whatsapp/client"
	"github.com/webitel/im-providers-service/internal/whatsapp/media"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type Contact struct {
	ID  uuid.UUID `json:"id" db:"id"`
	Iss string    `json:"iss" db:"iss"`
	Sub string    `json:"sub" db:"sub"`
}

func (contact *Contact) GetSub() string   { return contact.Sub }
func (contact *Contact) GetIss() string   { return contact.Iss }
func (contact *Contact) GetID() uuid.UUID { return contact.ID }

type WhatsappBusinessAccount struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	DC                   int        `json:"dc" db:"dc"`
	PhoneNumber          string     `json:"phone_number" db:"phone_number"`
	PhoneNumberID        string     `json:"phone_number_id" db:"phone_number_id"`
	AccessTokenEncrypted []byte     `json:"access_token_encrypted" db:"access_token_encrypted"`
	AccessTokenDecrypted string     `json:"-" db:"-"`
	AccessTokenExpiresAt *time.Time `json:"access_token_expires_at" db:"access_token_expires_at"`
	BusinessID           string     `json:"business_id" db:"business_id"`
	Bot                  Contact    `json:"bot" db:"bot"`
}

func (whatsappBusinessAccount *WhatsappBusinessAccount) DeepClone() WhatsappBusinessAccount {
	if whatsappBusinessAccount == nil {
		return WhatsappBusinessAccount{}
	}

	var expires *time.Time
	if whatsappBusinessAccount.AccessTokenExpiresAt != nil {
		expiresValue := *whatsappBusinessAccount.AccessTokenExpiresAt
		expires = &expiresValue
	}

	account := WhatsappBusinessAccount{
		ID:                   whatsappBusinessAccount.ID,
		PhoneNumber:          whatsappBusinessAccount.PhoneNumber,
		PhoneNumberID:        whatsappBusinessAccount.PhoneNumberID,
		AccessTokenEncrypted: whatsappBusinessAccount.AccessTokenEncrypted,
		AccessTokenDecrypted: whatsappBusinessAccount.AccessTokenDecrypted,
		AccessTokenExpiresAt: expires,
		BusinessID:           whatsappBusinessAccount.BusinessID,
		Bot:                  whatsappBusinessAccount.Bot,
		DC:                   whatsappBusinessAccount.DC,
	}

	return account
}

func (whatsappBusinessAccount *WhatsappBusinessAccount) PostFetch(encryptor Encryptor) (WhatsappBusinessAccount, error) {
	if whatsappBusinessAccount == nil {
		return WhatsappBusinessAccount{}, errors.InvalidArgument("received nil pointer whatsapp business account caller", errors.WithID("common.model.post_fetch"))
	}

	if encryptor == nil {
		return WhatsappBusinessAccount{}, errors.InvalidArgument("received nil pointer ecnryptor instance", errors.WithID("common.model.post_fetch"))
	}

	clone := whatsappBusinessAccount.DeepClone()
	if clone.AccessTokenDecrypted != "" {
		return clone, nil
	}

	decryptedAccessToken, err := encryptor.Decrypt(string(clone.AccessTokenEncrypted))
	if err != nil {
		return WhatsappBusinessAccount{}, errors.Internal("decrypting business account access token", errors.WithCause(err), errors.WithID("common.model.post_fetch"))
	}
	clone.AccessTokenDecrypted = decryptedAccessToken

	return clone, nil
}

func (whatsappBusinessAccount *WhatsappBusinessAccount) IsTokenExpires() bool {
	if whatsappBusinessAccount.AccessTokenExpiresAt == nil {
		return false
	}

	return time.Now().UTC().After(whatsappBusinessAccount.AccessTokenExpiresAt.UTC())
}

func (whatsappBusinessAccount *WhatsappBusinessAccount) CreateRequestClient() (*client.RequestClient, error) {
	if whatsappBusinessAccount == nil {
		return nil, errors.InvalidArgument("received nil pointer whatsapp business account caller", errors.WithID("common.model.create_requiest_client"))
	}

	if whatsappBusinessAccount.AccessTokenDecrypted == "" {
		return nil, errors.InvalidArgument("received business account with empty decrypted access token", errors.WithID("common.model.create_requiest_client"))
	}

	if whatsappBusinessAccount.IsTokenExpires() {
		return nil, errors.InvalidArgument(
			"received business account with expired access token",
			errors.WithID("common.model.create_requiest_client"),
			errors.WithValue("phone_number_id", whatsappBusinessAccount.PhoneNumberID),
			errors.WithValue("expires_at", whatsappBusinessAccount.AccessTokenExpiresAt),
		)
	}

	requestClient, err := client.NewRequesClient(client.WithAccessTokenConfig(whatsappBusinessAccount.AccessTokenDecrypted))
	if err != nil {
		return nil, errors.Internal("creating whatsapp business account request client", errors.WithCause(err), errors.WithID("common.model.create_requiest_client"))
	}

	return requestClient, nil
}

func (whatsappBusinessAccount *WhatsappBusinessAccount) CreateMediaClient() (*media.MediaManager, error) {
	if whatsappBusinessAccount == nil {
		return nil, errors.InvalidArgument("received nil pointer whatsapp business account caller", errors.WithID("whatsapp.common.model.create_media_client"))
	}

	requestClient, err := whatsappBusinessAccount.CreateRequestClient()
	if err != nil {
		return nil, errors.Wrap(err, errors.WithID("whatsapp.common.model.create_media_client"))
	}

	mediaClient := media.NewMediaManager(*requestClient)

	return mediaClient, nil
}
