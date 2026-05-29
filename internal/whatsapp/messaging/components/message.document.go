package components

import (
	"encoding/json"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type DocumentMessage struct {
	ID       string  `json:"id,omitempty"`
	Link     string  `json:"link,omitempty"`
	Caption  *string `json:"caption,omitempty"`
	FileName string  `json:"filename"`
}

func (documentMessage *DocumentMessage) Validate() error {
	if documentMessage == nil {
		return errors.InvalidArgument("document message is required", errors.WithID("message.document.validate"))
	}
	if documentMessage.FileName == "" || strings.Trim(documentMessage.FileName, " ") == "" {
		return errors.InvalidArgument("document message file name is required", errors.WithID("message.document.validate"))
	}
	return nil
}

type DocumentMessageApiPayload struct {
	BaseMessagePayload `json:",inline"`

	Document DocumentMessage `json:"document"`
}

type DocumentMessageConfigs struct {
	ID       string  `json:"id"`
	Link     string  `json:"link"`
	Caption  *string `json:"caption,omitempty"`
	FileName string  `json:"file_name"`
}

func (configs *DocumentMessageConfigs) Validate() error {
	if configs == nil {
		return errors.InvalidArgument("document message configs is required", errors.WithID("message.document.validate"))
	}

	if (configs.ID == "" || strings.Trim(configs.ID, " ") == "") && (configs.Link == "" || strings.Trim(configs.Link, " ") == "") {
		return errors.InvalidArgument("file id or link is required", errors.WithID("message.document.validate"))
	}

	if configs.FileName == "" || strings.Trim(configs.FileName, " ") == "" {
		return errors.InvalidArgument("file name is required", errors.WithID("message.document.validate"))
	}

	return nil
}

func NewDocumentMessage(configs DocumentMessageConfigs) (*DocumentMessage, error) {
	if err := configs.Validate(); err != nil {
		return nil, err
	}

	return &DocumentMessage{
		ID:       configs.ID,
		Link:     configs.Link,
		Caption:  configs.Caption,
		FileName: configs.FileName,
	}, nil
}

func (documentMessage *DocumentMessage) ToJson(configs ApiCompatibleJsonConverterConfigs) ([]byte, error) {
	if err := configs.Validate(); err != nil {
		return nil, errors.InvalidArgument("validating ToJson document message configs", errors.WithCause(err), errors.WithID("message.document.to_json"), errors.WithValue("to_phone_number", configs.SendToPhoneNumber()))
	}

	jsonData := DocumentMessageApiPayload{
		BaseMessagePayload: CreateBaseMessagePayload(configs.SendToPhoneNumber(), MessageTypeDocument),
		Document: DocumentMessage{
			ID:       documentMessage.ID,
			Link:     documentMessage.Link,
			Caption:  documentMessage.Caption,
			FileName: documentMessage.FileName,
		},
	}

	if configs.ReplyToMessageID() != "" {
		jsonData.MessageContext = &MessageContext{
			MessageID: configs.ReplyToMessageID(),
		}
	}

	raw, err := json.Marshal(jsonData)
	if err != nil {
		return nil, errors.Internal("marshaling document message", errors.WithCause(err), errors.WithID("message.document.to_json"), errors.WithValue("to_phone_number", configs.SendToPhoneNumber()))
	}

	return raw, nil
}
