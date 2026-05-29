package components

import (
	"encoding/json"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type textMessage struct {
	Text         string
	AllowPreview bool
}

func (textMessage *textMessage) SetText(text string) { textMessage.Text = text }

type TextMessageConfigs struct {
	Text         string `json:"text"`
	AllowPreview bool   `json:"allowPreview,omitempty"`
}

func (configs *TextMessageConfigs) Validate() error {
	if configs == nil {
		return errors.InvalidArgument("configs is required", errors.WithID("message.text.validate"))
	}

	if configs.Text == "" || strings.Trim(configs.Text, " ") == "" {
		return errors.InvalidArgument("text is required", errors.WithID("message.text.validate"))
	}

	return nil
}

type TextMessageApiPayloadText struct {
	Body         string `json:"body"`
	AllowPreview bool   `json:"preview_url,omitempty"`
}

type TextMessageApiPayload struct {
	BaseMessagePayload `json:",inline"`
	Text               TextMessageApiPayloadText `json:"text"`
}

func NewTextMessage(configs TextMessageConfigs) (*textMessage, error) {
	if err := configs.Validate(); err != nil {
		return nil, err
	}

	textMessage := textMessage{
		Text:         configs.Text,
		AllowPreview: configs.AllowPreview,
	}

	return &textMessage, nil
}

func (message *textMessage) ToJson(configs ApiCompatibleJsonConverterConfigs) ([]byte, error) {
	if err := configs.Validate(); err != nil {
		return nil, err
	}

	jsonData := TextMessageApiPayload{
		BaseMessagePayload: CreateBaseMessagePayload(configs.SendingPhoneNumber, MessageTypeText),
		Text: TextMessageApiPayloadText{
			Body:         message.Text,
			AllowPreview: message.AllowPreview,
		},
	}

	marshalled, err := json.Marshal(jsonData)
	if err != nil {
		return nil, errors.Internal("marshaling text message payload", errors.WithCause(err), errors.WithID("message.text.to_json"), errors.WithValue("to_phone_number", configs.SendingPhoneNumber), errors.WithValue("body", message.Text))
	}

	return marshalled, nil
}
