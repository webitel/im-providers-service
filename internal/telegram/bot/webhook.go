package bot

import (
	"context"
	"fmt"
	"math"
	"mime"
	"path"
	"strconv"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

var (
	defaultTelegramIssuer = "telegram"
)

func (p *Provider) handleNewMessage(ctx context.Context, gate *model.Gate, msg *model.Message) error {
	if gate == nil {
		return errors.Internal("gate required to handle message")
	}

	if msg == nil {
		return errors.Internal("message requred to handle message")
	}

	var (
		err    error
		handle func(context.Context, *model.Gate, *model.Message) error
	)
	if msg.Document != nil {
		handle = p.handleDocument
	} else if len(msg.Photo) != 0 {
		handle = p.handlePhoto
	} else if msg.Location != nil {
		handle = p.handleLocation
	} else if msg.Contact != nil {
		handle = p.handleContact
	} else if msg.Text != nil {
		handle = p.handleTextMessage
	}

	if handle != nil {
		err = handle(ctx, gate, msg)
	}
	return err
}

func (p *Provider) getMessageRecipients(ctx context.Context, gate *model.Gate, msg *model.Message) (from *coremodel.Peer, to *coremodel.Peer, err error) {
	if msg.From == nil {
		return nil, nil, errors.Internal("message from required to handle message")
	}
	if gate == nil {
		return nil, nil, errors.Internal("gate required to handle message")
	}

	from, err = p.constructFrom(ctx, gate, msg.From)
	if err != nil {
		return nil, nil, err
	}

	to, err = p.constructTo(ctx, gate)
	if err != nil {
		return nil, nil, err
	}

	return from, to, nil

}

func (p *Provider) handleTextMessage(ctx context.Context, gate *model.Gate, msg *model.Message) error {
	from, to, err := p.getMessageRecipients(ctx, gate, msg)
	if err != nil {
		return err
	}

	if msg.Text == nil {
		return errors.Internal("message text required to handle message")
	}

	var (
		coreMessage = &coremodel.SendTextRequest{
			DomainID:   gate.DC,
			From:       *from,
			To:         *to,
			Body:       ParseTelegramMarkdown(*msg.Text, msg.Entities),
			ExternalID: strconv.FormatInt(msg.MessageID, 10),
		}
	)
	if msg.ReplyToMessage != nil {
		coreMessage.ReplyToExternalID = strconv.FormatInt(msg.ReplyToMessage.MessageID, 10)
	}

	coreMessage.ForwardOrigin = mapForwardOrigin(msg.ForwardOrigin)

	_, err = p.coreMessengerFor(gate).SendText(ctx, coreMessage)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) handleDocument(ctx context.Context, gate *model.Gate, msg *model.Message) error {
	if msg.Document == nil {
		return errors.Internal("document required to handle message")
	}

	from, to, err := p.getMessageRecipients(ctx, gate, msg)
	if err != nil {
		return err
	}

	externalDoc := msg.Document

	media, err := p.syncFile(ctx, gate, externalDoc.FileID, deref(externalDoc.FileName), deref(externalDoc.MimeType), externalDoc.FileSize)
	if err != nil {
		return err
	}

	var (
		text        = msg.Text
		coreMessage = &coremodel.SendDocumentRequest{
			DomainID: gate.DC,
			From:     *from,
			To:       *to,
			Document: coremodel.DocumentRequest{
				Documents: []*coremodel.Document{
					{
						ID:       media.id,
						FileName: media.name,
						MimeType: media.mimeType,
						Size:     media.size,
					},
				},
			},
			ExternalID: strconv.FormatInt(msg.MessageID, 10),
		}
	)
	if msg.ReplyToMessage != nil {
		coreMessage.ReplyToExternalID = strconv.FormatInt(msg.ReplyToMessage.MessageID, 10)
	}
	if msg.Text != nil {
		coreMessage.Document.Body = *text
	}

	coreMessage.ForwardOrigin = mapForwardOrigin(msg.ForwardOrigin)

	_, err = p.coreMessengerFor(gate).SendDocument(ctx, coreMessage)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) handlePhoto(ctx context.Context, gate *model.Gate, msg *model.Message) error {
	if len(msg.Photo) == 0 {
		return errors.Internal("photo required to handle message")
	}

	from, to, err := p.getMessageRecipients(ctx, gate, msg)
	if err != nil {
		return err
	}

	var (
		text        = msg.Text
		coreMessage = &coremodel.SendImageRequest{
			DomainID:   gate.DC,
			From:       *from,
			To:         *to,
			Image:      coremodel.ImageRequest{},
			ExternalID: strconv.FormatInt(msg.MessageID, 10),
		}
	)
	if msg.Text != nil {
		coreMessage.Image.Body = *text
	}
	if msg.ReplyToMessage != nil {
		coreMessage.ReplyToExternalID = strconv.FormatInt(msg.ReplyToMessage.MessageID, 10)
	}

	coreMessage.ForwardOrigin = mapForwardOrigin(msg.ForwardOrigin)

	// msg.Photo lists the available sizes of one photo, not several photos.
	// https://core.telegram.org/bots/api#message
	photo := largestPhotoSize(msg.Photo)

	media, err := p.syncFile(ctx, gate, photo.FileID, "", photoMimeType, photo.FileSize)
	if err != nil {
		return err
	}

	coreMessage.Image.Images = append(coreMessage.Image.Images, &coremodel.Image{
		ID:       media.id,
		FileName: media.name,
		MimeType: media.mimeType,
		Size:     media.size,
	})

	_, err = p.coreMessengerFor(gate).SendImage(ctx, coreMessage)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) handleLocation(ctx context.Context, gate *model.Gate, msg *model.Message) error {
	if msg.Location == nil {
		return errors.Internal("location required to handle message")
	}
	from, to, err := p.getMessageRecipients(ctx, gate, msg)
	if err != nil {
		return err
	}
	if gate.DC < int64(math.MinInt) || gate.DC > int64(math.MaxInt) {
		return fmt.Errorf("value of domain %d overflows int range", gate.DC)
	}
	var (
		coreMessage = &coremodel.SendLocationRequest{
			DomainID:      int(gate.DC), // dangerous
			From:          *from,
			To:            *to,
			Latitude:      msg.Location.Latitude,
			Longitude:     msg.Location.Longitude,
			ExternalID:    strconv.FormatInt(msg.MessageID, 10),
			ForwardOrigin: mapForwardOrigin(msg.ForwardOrigin),
		}
	)

	_, err = p.coreMessengerFor(gate).SendLocation(ctx, coreMessage)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) handleContact(ctx context.Context, gate *model.Gate, msg *model.Message) error {
	if msg.Contact == nil {
		return errors.Internal("contact required to handle message")
	}

	from, to, err := p.getMessageRecipients(ctx, gate, msg)
	if err != nil {
		return err
	}
	if gate.DC < int64(math.MinInt) || gate.DC > int64(math.MaxInt) {
		return fmt.Errorf("value of domain %d overflows int range", gate.DC)
	}
	_, err = p.coreMessengerFor(gate).SendContact(ctx, &coremodel.SendContactRequest{
		From:        *from,
		To:          *to,
		Name:        &msg.Contact.FirstName,
		PhoneNumber: &msg.Contact.PhoneNumber,
		Metadata: map[string]any{
			"user_id":   msg.Contact.UserID,
			"vcard":     msg.Contact.Vcard,
			"last_name": msg.Contact.LastName,
		},
		ExternalID:    strconv.FormatInt(msg.MessageID, 10),
		DomainID:      int(gate.DC),
		ForwardOrigin: mapForwardOrigin(msg.ForwardOrigin),
	})
	if err != nil {
		return err
	}

	return nil
}

// Telegram re-encodes every photo it delivers as JPEG.
const photoMimeType = "image/jpeg"

type syncedFile struct {
	id       string
	name     string
	mimeType string
	size     int64
}

// syncFile copies an inbound Telegram file into storage. The Bot API download
// URL carries the bot token, so the file is fetched here and core only ever
// receives the storage file id.
func (p *Provider) syncFile(ctx context.Context, gate *model.Gate, fileID, name, mimeType string, size *int64) (*syncedFile, error) {
	body, file, err := p.tgMessageClient.DownloadFile(ctx, gate.Token, fileID)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	if name == "" && file.FilePath != nil {
		name = path.Base(*file.FilePath)
	}

	if name == "" {
		name = fileID
	}

	if mimeType == "" {
		mimeType = mime.TypeByExtension(path.Ext(name))
	}

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	uploaded, err := p.mediaManager.UploadFile(ctx, coremodel.UploadRequest{
		DomainID: gate.DC,
		Name:     name,
		MimeType: mimeType,
	}, body)
	if err != nil {
		return nil, err
	}

	synced := &syncedFile{id: uploaded.ID, name: name, mimeType: mimeType, size: uploaded.Size}
	if size != nil && *size > 0 {
		synced.size = *size
	} else if synced.size <= 0 && file.FileSize != nil {
		synced.size = *file.FileSize
	}

	return synced, nil
}

func largestPhotoSize(sizes []model.PhotoSize) model.PhotoSize {
	largest := sizes[0]
	for _, s := range sizes[1:] {
		if s.Width*s.Height > largest.Width*largest.Height {
			largest = s
		}
	}

	return largest
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func mapForwardOrigin(origin *model.MessageOrigin) *coremodel.ForwardOrigin {
	if origin == nil {
		return nil
	}

	var kind coremodel.ForwardOriginKind

	switch origin.Type {
	case model.MessageOriginTypeUser:
		kind = coremodel.ForwardOriginExternalUser
	case model.MessageOriginTypeHiddenUser:
		kind = coremodel.ForwardOriginExternalHiddenUser
	case model.MessageOriginTypeChat, model.MessageOriginTypeChannel:
		kind = coremodel.ForwardOriginExternalChat
	default:
		kind = coremodel.ForwardOriginExternalHiddenUser
	}

	return &coremodel.ForwardOrigin{
		Kind:           kind,
		SenderName:     origin.SenderName(),
		OriginalSentAt: origin.OriginalSentAtMillis(),
	}
}
