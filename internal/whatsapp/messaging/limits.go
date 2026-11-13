package messaging

import (
	"github.com/webitel/im-providers-service/internal/core/model"
)

// Cloud API media caps for the kinds this channel sends: images go out as
// "image", every other file as "document" (video/audio/sticker are never used
// as outbound types here).
// https://developers.facebook.com/docs/whatsapp/cloud-api/reference/media#supported-media-types
const (
	maxImageBytes    = 5 << 20
	maxDocumentBytes = 100 << 20
)

func checkImageSize(images []*model.Image) error {
	if len(images) == 0 || images[0] == nil {
		return nil
	}

	return model.CheckFileSize("whatsapp image", images[0].Size, maxImageBytes)
}

func checkDocumentSize(docs []*model.Document) error {
	if len(docs) == 0 || docs[0] == nil {
		return nil
	}

	return model.CheckFileSize("whatsapp document", docs[0].Size, maxDocumentBytes)
}
