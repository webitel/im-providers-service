package events

import "github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"

type ContactMessageEvent struct {
	BaseMessageEvent `json:",inline"`
	Contacts         components.ContactMessage `json:"contacts"`
}

func NewContactsMessageEvent(baseMessageEvent BaseMessageEvent, contacts components.ContactMessage) *ContactMessageEvent {
	return &ContactMessageEvent{
		BaseMessageEvent: baseMessageEvent,
		Contacts:         contacts,
	}
}
