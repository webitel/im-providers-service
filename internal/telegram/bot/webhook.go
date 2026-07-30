package bot

import (
	"context"
	"fmt"
	"math"
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
			Body:       *msg.Text,
			ExternalID: strconv.FormatInt(msg.MessageID, 10),
		}
	)
	if msg.ReplyToMessage != nil {
		coreMessage.ReplyToExternalID = strconv.FormatInt(msg.ReplyToMessage.MessageID, 10)
	}
	_, err = p.coreMessageClient.SendText(ctx, coreMessage)
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

	docURL, err := p.tgMessageClient.GenerateFileURL(ctx, gate.Token, msg.Document.FileID)
	if err != nil {
		return err
	}

	var (
		text        = msg.Text
		externalDoc = msg.Document
		coreMessage = &coremodel.SendDocumentRequest{
			DomainID: gate.DC,
			From:     *from,
			To:       *to,
			Document: coremodel.DocumentRequest{
				Documents: []*coremodel.Document{
					{
						FileName: *externalDoc.FileName,
						MimeType: *externalDoc.MimeType,
						Size:     *externalDoc.FileSize,
						URL:      docURL.String(),
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

	_, err = p.coreMessageClient.SendDocument(ctx, coreMessage)
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

	for _, image := range msg.Photo {
		docURL, err := p.tgMessageClient.GenerateFileURL(ctx, gate.Token, image.FileID)
		if err != nil {
			return err
		}
		coreMessage.Image.Images = append(coreMessage.Image.Images, &coremodel.Image{
			URL:  docURL.String(),
			Size: *image.FileSize,
		})
	}

	_, err = p.coreMessageClient.SendImage(ctx, coreMessage)
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
			DomainID:   int(gate.DC), // dangerous
			From:       *from,
			To:         *to,
			Latitude:   msg.Location.Latitude,
			Longitude:  msg.Location.Longitude,
			ExternalID: strconv.FormatInt(msg.MessageID, 10),
		}
	)

	_, err = p.coreMessageClient.SendLocation(ctx, coreMessage)
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
	_, err = p.coreMessageClient.SendContact(ctx, &coremodel.SendContactRequest{
		From:        *from,
		To:          *to,
		Name:        &msg.Contact.FirstName,
		PhoneNumber: &msg.Contact.PhoneNumber,
		Metadata: map[string]any{
			"user_id":   msg.Contact.UserID,
			"vcard":     msg.Contact.Vcard,
			"last_name": msg.Contact.LastName,
		},
		ExternalID: strconv.FormatInt(msg.MessageID, 10),
		DomainID:   int(gate.DC),
	})
	if err != nil {
		return err
	}

	return nil
}
