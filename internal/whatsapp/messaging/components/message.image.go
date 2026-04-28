package components

import (
	"encoding/json"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type ImageMessage struct {
	ID      string  `json:"id,omitempty"`
	Link    string  `json:"link,omitempty"`
	Caption *string `json:"caption,omitempty"`
}

func (imageMessage *ImageMessage) Validate() error {
	if imageMessage == nil {
		return errors.InvalidArgument("image message is required", errors.WithID("message.image.validate"))
	}
	if (imageMessage.ID == "" || strings.Trim(imageMessage.ID, " ") == "") && (imageMessage.Link == "" || strings.Trim(imageMessage.Link, " ") == "") {
		return errors.InvalidArgument("message.image.validate", errors.WithID("message.image.validate"))
	}
	return nil
}

type ImageMessageApiPayload struct {
	BaseMessagePayload `json:",inline"`

	Image ImageMessage `json:"image"`
}

type ImageMessageConfigs struct {
	ID      string  `json:"id"`
	Link    string  `json:"link"`
	Caption *string `json:"caption"`
}

func (configs *ImageMessageConfigs) Validate() error {
	if configs == nil {
		return errors.InvalidArgument("image message configs is required", errors.WithID("message.image.validate"))
	}
	if configs.ID == "" && configs.Link == "" {
		return errors.InvalidArgument("image message require image link or ID", errors.WithID("message.image.validate"))
	}
	return nil
}

func NewImageMessage(configs ImageMessageConfigs) (*ImageMessage, error) {
	if err := configs.Validate(); err != nil {
		return nil, err
	}

	imageMessage := ImageMessage{
		ID:      configs.ID,
		Link:    configs.Link,
		Caption: configs.Caption,
	}

	return &imageMessage, nil
}

func (imageMessage *ImageMessage) ToJson(configs ApiCompatibleJsonConverterConfigs) ([]byte, error) {
	if err := configs.Validate(); err != nil {
		return nil, errors.InvalidArgument("validating image message ToJson configs", errors.WithCause(err), errors.WithID("message.image.to_json"))
	}

	jsonData := ImageMessageApiPayload{
		BaseMessagePayload: CreateBaseMessagePayload(configs.SendToPhoneNumber(), MessageTypeImage),
		Image: ImageMessage{
			ID:      imageMessage.ID,
			Link:    imageMessage.Link,
			Caption: imageMessage.Caption,
		},
	}

	if configs.ReplyToMessageID() != "" {
		jsonData.MessageContext = &MessageContext{
			MessageID: configs.ReplyToMessageID(),
		}
	}

	raw, err := json.Marshal(jsonData)
	if err != nil {
		return nil, errors.Internal("marshaling image message json data", errors.WithCause(err), errors.WithID("message.image.to_json"), errors.WithValue("to_phone_number", configs.SendToPhoneNumber()))
	}

	return raw, nil
}
