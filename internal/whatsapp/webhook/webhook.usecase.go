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

type WhatsAppBusinessAccountResolver interface {
	Resolve(ctx context.Context, query WhatsAppBusinessAccountResolveQuery) (*common.WhatsappBusinessAccount, error)
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
	whatsAppBusinessAccount, err := webhook.whatsAppBusinessAccountResolver.Resolve(ctx, WhatsAppBusinessAccountResolveQuery{PhoneNumberID: phoneNumberID})
	if err != nil {
		if errors.Is(err, WebhookErrDisablled) {
			return nil, errors.Wrap(err, errors.WithID("webhook.usecase.resolve_whatsapp_business_account"))
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

	//TODO: add external message corelation ID pass
	coreTextMessage := model.SendTextRequest{
		From: model.Peer{
			Type: model.PeerUser,
			Sub:  whatsAppBusinessAccount.Bot.Sub,
			Iss:  whatsAppBusinessAccount.Bot.Iss,
		},
		To: model.Peer{
			Type: model.PeerUser,
			Sub:  "whatsapp",
			Iss:  textEvent.From,
		},
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
