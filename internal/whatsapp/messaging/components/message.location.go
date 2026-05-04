package components

import (
	"encoding/json"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type LocationMessage struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
	Name      string  `json:"name,omitempty"`
}

type LocationMessageAPIPayload struct {
	BaseMessagePayload
	Location LocationMessage `json:"location"`
}

func NewLocationMessage(latitude, logitude float64) *LocationMessage {
	return &LocationMessage{
		Latitude:  latitude,
		Longitude: logitude,
	}
}

func (locationMessage *LocationMessage) SetAddress(address string) *LocationMessage {
	locationMessage.Address = address
	return locationMessage
}

func (locationMessage *LocationMessage) SetName(name string) *LocationMessage {
	locationMessage.Name = name
	return locationMessage
}

func (locationMessage *LocationMessage) ToJSON(configs ApiCompatibleJsonConverterConfigs) ([]byte, error) {
	jsonData := LocationMessageAPIPayload{
		BaseMessagePayload: CreateBaseMessagePayload(configs.SendToPhoneNumber(), MessageTypeLocation),
		Location:           *locationMessage,
	}

	if configs.ReplyToMessageID() != "" {
		jsonData.MessageContext = &MessageContext{
			MessageID: configs.ReplyToMessageID(),
		}
	}

	raw, err := json.Marshal(jsonData)
	if err != nil {
		return nil, errors.Internal("marshaling location message api payload", errors.WithCause(err), errors.WithID("whatsapp.components.message_location"))
	}

	return raw, nil
}
