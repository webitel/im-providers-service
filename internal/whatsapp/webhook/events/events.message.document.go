package events

import "github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"

type DocumentMessageEvent struct {
	BaseMediaMessageEvent
	Document components.DocumentMessage
}

func NewDocumentMessageEvent(baseMessageEvent BaseMessageEvent, document components.DocumentMessage, mediaID, sha256, mimeType string) *DocumentMessageEvent {
	return &DocumentMessageEvent{
		BaseMediaMessageEvent: BaseMediaMessageEvent{
			BaseMessageEvent: baseMessageEvent,
			MediaID:          mediaID,
			MimeType:         mimeType,
			Sha256:           sha256,
		},
		Document: document,
	}
}
