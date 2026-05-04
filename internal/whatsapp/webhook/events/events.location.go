package events

import "github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"

type LocationMessageEvent struct {
	BaseMessageEvent
	Location components.LocationMessage `json:"location"`
}

func NewLocationMessageEvent(baseMessageEvent BaseMessageEvent, location components.LocationMessage) *LocationMessageEvent {
	return &LocationMessageEvent{
		BaseMessageEvent: baseMessageEvent,
		Location:         location,
	}
}
