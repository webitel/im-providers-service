package components

import (
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeDocument MessageType = "document"
	MessageTypeImage    MessageType = "image"
)

type ApiCompatibleJsonConverterConfigs struct {
	ReplyToMessage     string
	SendingPhoneNumber string
}

func (configs *ApiCompatibleJsonConverterConfigs) Validate() error {
	if configs == nil {
		return errors.InvalidArgument("configs is required", errors.WithID("message.base.validate"))
	}
	if configs.SendingPhoneNumber == "" || strings.Trim(configs.SendingPhoneNumber, " ") == "" {
		return errors.InvalidArgument("sending phone number is required", errors.WithID("message.base.validate"))
	}

	return nil
}

func (apiCompatibleJsonConverterConfigs *ApiCompatibleJsonConverterConfigs) ReplyToMessageID() string {
	return apiCompatibleJsonConverterConfigs.ReplyToMessage
}

func (apiCompatibleJsonConverterConfigs *ApiCompatibleJsonConverterConfigs) SendToPhoneNumber() string {
	return apiCompatibleJsonConverterConfigs.SendingPhoneNumber
}

type MessageContext struct {
	MessageID string `json:"message_id,omitempty"`
}

type BaseMessagePayload struct {
	MessageContext   *MessageContext `json:"message_context,omitempty"`
	To               string          `json:"to"`
	Type             MessageType     `json:"type"`
	MessagingProduct string          `json:"messaging_product"`
	RecipientType    string          `json:"recipient_type"`
}

func CreateBaseMessagePayload(to string, messageType MessageType) BaseMessagePayload {
	return BaseMessagePayload{
		To:               to,
		Type:             messageType,
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
	}
}
