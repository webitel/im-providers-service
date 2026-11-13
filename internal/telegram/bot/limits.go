package bot

import (
	coremodel "github.com/webitel/im-providers-service/internal/core/model"
)

// Bot API limits for files sent by HTTP URL, the only way this provider sends
// media. Telegram downloads the file itself and answers an oversized one with
// a generic fetch error, so the size must be checked here to report it.
// https://core.telegram.org/bots/api#sending-files
const (
	maxPhotoURLBytes    = 5 << 20
	maxDocumentURLBytes = 20 << 20
)

// checkImageSizes rejects the whole message before the first send, so an
// oversized N-th file does not leave earlier files delivered.
func checkImageSizes(images []*coremodel.Image) error {
	for _, img := range images {
		if img == nil {
			continue
		}

		if err := coremodel.CheckFileSize("telegram photo", img.Size, maxPhotoURLBytes); err != nil {
			return err
		}
	}

	return nil
}

func checkDocumentSizes(docs []*coremodel.Document) error {
	for _, doc := range docs {
		if doc == nil {
			continue
		}

		if err := coremodel.CheckFileSize("telegram document", doc.Size, maxDocumentURLBytes); err != nil {
			return err
		}
	}

	return nil
}
