package webhook

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/im-providers-service/internal/whatsapp/webhook/events"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type MediaUploader interface {
	UploadFile(ctx context.Context, uploadMetadata model.UploadRequest, body io.Reader) (model.UploadResponse, error)
}

type CoreMessanger interface {
	SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error)
	SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error)
	SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error)
	SendContact(ctx context.Context, in *model.SendContactRequest) (*model.SendResponse, error)
	SendLocation(ctx context.Context, in *model.SendLocationRequest) (*model.SendResponse, error)
}

type WhatsAppBusinessAccountResolveQuery struct {
	PhoneNumberID string
}

func (whatsAppBusinessAccountResolveQuery *WhatsAppBusinessAccountResolveQuery) GetPhoneNumberID() string {
	return whatsAppBusinessAccountResolveQuery.PhoneNumberID
}

type WhatsAppBusinessAccountResolver interface {
	Resolve(ctx context.Context, query *WhatsAppBusinessAccountResolveQuery) (*common.WhatsappBusinessAccount, error)
}

type webhook struct {
	logger                          *slog.Logger
	coreMessanger                   CoreMessanger
	whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver
	encryptor                       common.Encryptor
	mediaUploader                   MediaUploader
}

func newWebhook(
	logger *slog.Logger,
	coreMessanger CoreMessanger,
	whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver,
	encryptor common.Encryptor,
	mediaUploader MediaUploader,
) *webhook {
	log := logger.With("component", "whatsapp_webhook_usecase")
	return &webhook{
		logger:                          log,
		coreMessanger:                   coreMessanger,
		whatsAppBusinessAccountResolver: whatsAppBusinessAccountResolver,
		encryptor:                       encryptor,
		mediaUploader:                   mediaUploader,
	}
}

func (webhook *webhook) resolveWhatsappBusinessAccount(ctx context.Context, phoneNumberID string) (*common.WhatsappBusinessAccount, error) {
	whatsAppBusinessAccount, err := webhook.whatsAppBusinessAccountResolver.Resolve(ctx, &WhatsAppBusinessAccountResolveQuery{PhoneNumberID: phoneNumberID})
	if err != nil {
		if errors.Is(err, WebhookErrDisablled) {
			return nil, nil
		}
		return nil, errors.Wrap(err, errors.WithID("webhook.usecase.resolve_whatsapp_business_account"))
	}

	preparedBusinessAccount, err := whatsAppBusinessAccount.PostFetch(webhook.encryptor)
	if err != nil {
		return nil, errors.Internal("preparing whatsapp business account after fetch", errors.WithCause(err), errors.WithID("webhook.usecase.resolve_whatsapp_business_account"))
	}

	return &preparedBusinessAccount, nil
}

func (webhook *webhook) uploadReceivedMedia(ctx context.Context, metadata model.UploadRequest, whatsAppBusinessAccount *common.WhatsappBusinessAccount) (model.UploadResponse, error) {
	urlObj := metadata.URL
	mediaClient, err := whatsAppBusinessAccount.CreateMediaClient()
	if err != nil {
		return model.UploadResponse{}, errors.Wrap(err, errors.WithID("whatsapp.webhook.usecase.upload_received_media"), errors.WithValue("phone_number_id", whatsAppBusinessAccount.PhoneNumberID))
	}

	if urlObj == "" {
		if urlObj, err = mediaClient.GetMediaURLByID(ctx, metadata.ExternalID); err != nil {
			return model.UploadResponse{}, errors.Wrap(err, errors.WithID("whatsapp.webhook.usecase.upload_received_media"))
		}
	}

	file, mime, err := mediaClient.DownloadMediaByURL(ctx, urlObj) // add fallback to retrive URL by ID in case of 404
	if err != nil {
		return model.UploadResponse{}, errors.Wrap(err, errors.WithID("whatsapp.webhook.usecase.upload_received_media"))
	}
	defer file.Close()

	metadata.MimeType = mime

	uploadedMetadata, err := webhook.mediaUploader.UploadFile(ctx, metadata, file)
	if err != nil {
		return model.UploadResponse{}, errors.Wrap(err, errors.WithID("whatsapp.webhook.usecase.upload_received_media"))
	}

	return uploadedMetadata, nil
}

func (webhook *webhook) HandleTextMessage(ctx context.Context, textEvent *events.TextMessageEvent) error {
	log := webhook.logger.With("operation", "handle_text_message")

	if textEvent == nil {
		log.Warn("received nil pointer as text event")
		return errors.InvalidArgument("received nil pointer as text event", errors.WithID("webhook.usecase.handle_text_message"))
	}

	whatsAppBusinessAccount, err := webhook.resolveWhatsappBusinessAccount(ctx, textEvent.BaseMessageEvent.PhoneNumber.ID)
	if err != nil {
		log.Error("resolving whatsapp business account binded to gate", "error", err)
		return errors.Wrap(err, errors.WithID("webhook.usecase.handle_text_message"))
	}

	if whatsAppBusinessAccount == nil {
		return nil
	}

	//TODO: add external message corelation ID pass
	coreTextMessage := model.SendTextRequest{
		To:       extractPeerFromWhatsAppBusinessAccount(whatsAppBusinessAccount),
		From:     extractPeerFromWebhookInput(textEvent.From, textEvent.SenderName),
		Body:     textEvent.Text,
		DomainID: int64(whatsAppBusinessAccount.DC),
	}

	_, err = webhook.coreMessanger.SendText(ctx, &coreTextMessage)
	if err != nil {
		log.Error(
			"sending text message request to IM core",
			errors.WithCause(err),
			errors.WithID("webhook.usecase.handle_text_message"),
			errors.WithValue("from", textEvent.From),
			errors.WithValue("to", textEvent.PhoneNumber.ID),
		)
		return err
	}

	return nil
}

func extractPeerFromWhatsAppBusinessAccount(whatsAppBusinessAccount *common.WhatsappBusinessAccount) model.Peer {
	if whatsAppBusinessAccount == nil {
		return model.Peer{}
	}

	return model.Peer{
		Type: model.PeerUser,
		Sub:  whatsAppBusinessAccount.Bot.GetSub(),
		Iss:  whatsAppBusinessAccount.Bot.GetIss(),
	}
}

func extractPeerFromWebhookInput(senderPhoneNumber, senderName string) model.Peer {
	return model.Peer{
		Type: model.PeerUser,
		Sub:  senderPhoneNumber,
		Iss:  "whatsapp",
		Name: senderName,
	}
}

func (webhook *webhook) HandleDocumentMessage(ctx context.Context, documentEvent *events.DocumentMessageEvent) error {
	log := webhook.logger.With("operation", "handle_document_message")

	if documentEvent == nil {
		log.Warn("received nil pointer document event")
		return errors.InvalidArgument("received nil pointer document event", errors.WithID("whatsapp.webhook.usecase.handle_document_message"))
	}

	whatsAppBusinessAccount, err := webhook.resolveWhatsappBusinessAccount(ctx, documentEvent.PhoneNumber.ID)
	if err != nil {
		log.Error("resolving whatsapp business account", "error", err, "phone_number_id", documentEvent.PhoneNumber.ID)
		return err
	}

	if whatsAppBusinessAccount == nil {
		return nil
	}

	uploadedMd, err := webhook.uploadReceivedMedia(
		ctx,
		model.UploadRequest{
			DomainID:   int64(whatsAppBusinessAccount.DC),
			MimeType:   documentEvent.MimeType,
			Name:       documentEvent.Document.FileName,
			URL:        documentEvent.Document.Link,
			ExternalID: documentEvent.Document.ID,
		},
		whatsAppBusinessAccount,
	)

	if err != nil {
		log.Error("uploading received document to internal storage", "error", err)
		return err
	}

	coreDocumentMessage := model.SendDocumentRequest{
		From: extractPeerFromWebhookInput(documentEvent.From, documentEvent.SenderName),
		To:   extractPeerFromWhatsAppBusinessAccount(whatsAppBusinessAccount),
		Document: model.DocumentRequest{
			Body: *documentEvent.Document.Caption,
			Documents: []*model.Document{
				{
					FileName: documentEvent.Document.FileName,
					MimeType: documentEvent.MimeType,
					Size:     uploadedMd.Size,
					ID:       uploadedMd.ID,
				},
			},
		},
		DomainID: int64(whatsAppBusinessAccount.DC),
	}

	if _, err = webhook.coreMessanger.SendDocument(ctx, &coreDocumentMessage); err != nil {
		log.Error("sending document request to IM core", "error", err)
		return errors.Internal("sending document request to IM core", errors.WithCause(err), errors.WithID("whatsapp.webhook.usecase.handle_document_message"))
	}

	return nil
}

func (webhook *webhook) HandleImageMessage(ctx context.Context, imageEvent *events.ImageMessageEvent) error {
	log := webhook.logger.With("operation", "handle_image_message")

	if imageEvent == nil {
		log.Warn("received nil pointer image event")
		return errors.InvalidArgument("received nil pointer image event", errors.WithID("whatsapp.webhook.usecase.handle_image_message"))
	}

	whatsAppBusinessAccount, err := webhook.resolveWhatsappBusinessAccount(ctx, imageEvent.PhoneNumber.ID)
	if err != nil {
		log.Error("resolving whatsapp business account", "error", err)
		return err
	}

	if whatsAppBusinessAccount == nil {
		return nil
	}

	mediaMetadata, err := webhook.uploadReceivedMedia(
		ctx,
		model.UploadRequest{
			DomainID:   int64(whatsAppBusinessAccount.DC),
			MimeType:   imageEvent.MimeType,
			Name:       fmt.Sprintf("%s-%s", imageEvent.SenderName, time.Now().String()),
			URL:        imageEvent.Image.Link,
			ExternalID: imageEvent.MediaId,
		},
		whatsAppBusinessAccount,
	)

	if err != nil {
		log.Error("uploading received document to internal storage", "error", err)
		return err
	}

	coreImageMessage := model.SendImageRequest{
		From: extractPeerFromWebhookInput(imageEvent.From, imageEvent.SenderName),
		To:   extractPeerFromWhatsAppBusinessAccount(whatsAppBusinessAccount),
		Image: model.ImageRequest{
			Images: []*model.Image{
				{
					MimeType: imageEvent.MimeType,
					ID:       mediaMetadata.ID,
				},
			},
			Body: *imageEvent.Image.Caption,
		},
		DomainID: int64(whatsAppBusinessAccount.DC),
	}

	if _, err := webhook.coreMessanger.SendImage(ctx, &coreImageMessage); err != nil {
		log.Error("sending image request to IM core", "error", err, "from", imageEvent.From, "to", whatsAppBusinessAccount.Bot.Sub)
		return err
	}

	return nil
}

func (webhook *webhook) HandleLocationMessage(ctx context.Context, locationEvent *events.LocationMessageEvent) error {
	log := webhook.logger.With("operation", "handle_location_message")

	if locationEvent == nil {
		log.Warn("received nil pointer send location event")
		return errors.InvalidArgument("received nil pointer send location event", errors.WithID("whatsapp.webhook.usecase.handle_location_message"))
	}

	whatsappBusinessAccount, err := webhook.resolveWhatsappBusinessAccount(ctx, locationEvent.PhoneNumber.ID)
	if err != nil {
		log.Error("resolving whatsapp business account", "error", err)
		return errors.Wrap(err, errors.WithID("whatsapp.webhook.usecase.handle_location_message"))
	}

	if whatsappBusinessAccount == nil {
		return nil
	}

	var namePtr, addressPtr *string
	if locationEvent.Location.Name != "" {
		namePtr = &locationEvent.Location.Name
	}
	if locationEvent.Location.Address != "" {
		addressPtr = &locationEvent.Location.Address
	}

	locationMessage := model.SendLocationRequest{
		From:       extractPeerFromWebhookInput(locationEvent.From, locationEvent.SenderName),
		To:         extractPeerFromWhatsAppBusinessAccount(whatsappBusinessAccount),
		Latitude:   locationEvent.Location.Latitude,
		Longitude:  locationEvent.Location.Longitude,
		Name:       namePtr,
		Address:    addressPtr,
		ExternalID: locationEvent.MessageID,
		DomainID:   whatsappBusinessAccount.DC,
	}

	if _, err = webhook.coreMessanger.SendLocation(ctx, &locationMessage); err != nil {
		log.Error("sending location message to IM core", "error", err)
		return errors.Wrap(err, errors.WithID("whatsapp.webhook.usecase.handle_location_message"))
	}

	return nil
}

func (webhook *webhook) HandleContactsMessage(ctx context.Context, contacts *events.ContactMessageEvent) error {
	log := webhook.logger.With("operation", "handle_contacts_message")
	if contacts == nil {
		log.Warn("received nil pointer contacts event")
		return errors.InvalidArgument("contacts event is required", errors.WithID("whatsapp.webhook.usecase.handle_contacts_message"))
	}

	whatsappBusinessAccount, err := webhook.resolveWhatsappBusinessAccount(ctx, contacts.PhoneNumber.ID)
	if err != nil {
		log.Error("resolving whatsapp business account")
		return errors.Wrap(err, errors.WithID("whatsapp.webhook.usecase.handle_contacts_message"))
	}

	if whatsappBusinessAccount == nil {
		return nil
	}

	fromPeer := extractPeerFromWebhookInput(contacts.From, contacts.SenderName)
	toPeer := extractPeerFromWhatsAppBusinessAccount(whatsappBusinessAccount)

	for _, contact := range contacts.Contacts.Contacts { //TODO: add worker pool to send IM messages concurrently
		var namePtr, emailPtr, phoneNumberPtr *string
		if len(contact.Emails) > 0 {
			emailPtr = &contact.Emails[0].Email
		}

		if contact.Name.FormattedName != "" {
			namePtr = &contact.Name.FormattedName
		}

		if len(contact.Phones) > 0 {
			phoneNumberPtr = &contact.Phones[0].Phone
		}

		contactMessage := model.SendContactRequest{
			From:        fromPeer,
			To:          toPeer,
			Name:        namePtr,
			Email:       emailPtr,
			PhoneNumber: phoneNumberPtr,
			Metadata:    contact.AsMetadata(),
			ExternalID:  "",
			DomainID:    whatsappBusinessAccount.DC,
		}

		if _, err := webhook.coreMessanger.SendContact(ctx, &contactMessage); err != nil {
			log.Error("sending contact message to IM core", "error", err, "from", contacts.From, "phone_number_id", whatsappBusinessAccount.PhoneNumberID)
			return err
		}
	}

	return nil
}
