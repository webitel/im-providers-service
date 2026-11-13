package facebook

import (
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// maxAttachmentBytes is the Send API cap for any attachment sent by URL.
// https://developers.facebook.com/docs/messenger-platform/send-messages#sending_attachments
const maxAttachmentBytes = 25 << 20

type sizeGetter interface {
	GetSize() int64
}

// checkFirstSize checks the attachment the Send API call will carry: only
// the first item of a multi-file message is sent (see firstURL).
func checkFirstSize[T sizeGetter](what string, items []T) error {
	if len(items) == 0 {
		return nil
	}

	return sharedmodel.CheckFileSize(what, items[0].GetSize(), maxAttachmentBytes)
}
