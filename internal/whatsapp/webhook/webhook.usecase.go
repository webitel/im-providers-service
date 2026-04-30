package webhook

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
	"github.com/webitel/im-providers-service/internal/whatsapp/webhook/events"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type CoreMessanger interface {
	SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error)
	SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error)
	SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error)
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
}

func newWebhook(logger *slog.Logger, coreMessanger CoreMessanger, whatsAppBusinessAccountResolver WhatsAppBusinessAccountResolver, encryptor common.Encryptor) *webhook {
	log := logger.With("component", "whatsapp_webhook_usecase")
	return &webhook{logger: log, coreMessanger: coreMessanger, whatsAppBusinessAccountResolver: whatsAppBusinessAccountResolver, encryptor: encryptor}
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

	coreDocumentMessage := model.SendDocumentRequest{
		From: extractPeerFromWebhookInput(documentEvent.From, documentEvent.SenderName),
		To:   extractPeerFromWhatsAppBusinessAccount(whatsAppBusinessAccount),
		Document: model.DocumentRequest{
			Body: *documentEvent.Document.Caption,
			Documents: []*model.Document{
				{
					FileName: documentEvent.Document.FileName,
					MimeType: documentEvent.MimeType,
					Size:     0,
					URL:      documentEvent.Document.Link,
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

	coreImageMessage := model.SendImageRequest{
		From: extractPeerFromWebhookInput(imageEvent.From, imageEvent.SenderName),
		To:   extractPeerFromWhatsAppBusinessAccount(whatsAppBusinessAccount),
		Image: model.ImageRequest{
			Images: []*model.Image{
				{
					MimeType: imageEvent.MimeType,
					URL:      imageEvent.Image.Link,
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
