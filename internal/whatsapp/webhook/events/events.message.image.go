package events

import "github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"

type ImageMessageEvent struct {
	BaseMediaMessageEvent

	Image components.ImageMessage
}

func NewImageMessageEvent(baseMessageEvent BaseMessageEvent, image components.ImageMessage, mediaID, sha256, mimeType string) *ImageMessageEvent {
	return &ImageMessageEvent{
		BaseMediaMessageEvent: BaseMediaMessageEvent{
			BaseMessageEvent: baseMessageEvent,
			MediaID:          mediaID,
			MimeType:         mimeType,
			Sha256:           sha256,
		},
		Image: image,
	}
}
