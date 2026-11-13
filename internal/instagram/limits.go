package instagram

import (
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// maxImageBytes is the Instagram Messaging cap for an image attachment, the
// only media kind this channel sends (video/audio would be 25 MB).
// https://developers.facebook.com/docs/instagram-platform/instagram-api-with-instagram-login/messaging-api#send-a-media-message
const maxImageBytes = 8 << 20

func checkImageSize(images []*sharedmodel.Image) error {
	if len(images) == 0 || images[0] == nil {
		return nil
	}

	return sharedmodel.CheckFileSize("instagram image", images[0].Size, maxImageBytes)
}
