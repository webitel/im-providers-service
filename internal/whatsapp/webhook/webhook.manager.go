package webhook

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/whatsapp/client"
	"github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"
	"github.com/webitel/im-providers-service/internal/whatsapp/webhook/events"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type NoopWebhookSender struct{}

func (noopWebhookSender *NoopWebhookSender) Type() string { return "whatsapp" }
func (noopWebhookSender *NoopWebhookSender) SendText(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	return nil, nil
}
func (noopWebhookSender *NoopWebhookSender) SendImage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	return nil, nil
}
func (noopWebhookSender *NoopWebhookSender) SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	return nil, nil
}

type CoreIntegrationHandler interface {
	HandleTextMessage(ctx context.Context, textEvent *events.TextMessageEvent) error
	HandleDocumentMessage(ctx context.Context, documentEvent *events.DocumentMessageEvent) error
	HandleImageMessage(ctx context.Context, imageEvent *events.ImageMessageEvent) error
}

type WebhookManager struct {
	NoopWebhookSender

	secret                 string
	path                   string
	RequestClient          client.RequestClient
	logger                 *slog.Logger
	coreIntegrationHandler CoreIntegrationHandler
}

type WebhookManagerConfig struct {
	Secret        string
	Path          string
	RequestClient client.RequestClient
	Logger        *slog.Logger
}

func (webhookManagerConfig *WebhookManagerConfig) Validate() error {
	return nil
}

func newWebhookManager(webhookManagerConfig WebhookManagerConfig, coreIntegrationHandler CoreIntegrationHandler) (*WebhookManager, error) {
	if err := webhookManagerConfig.Validate(); err != nil {
		return nil, err
	}

	webhookManager := WebhookManager{
		secret:                 webhookManagerConfig.Secret,
		path:                   webhookManagerConfig.Path,
		RequestClient:          webhookManagerConfig.RequestClient,
		logger:                 webhookManagerConfig.Logger,
		coreIntegrationHandler: coreIntegrationHandler,
	}

	return &webhookManager, nil
}

func unmarshallWebhookValue[T any](payload any) (T, error) {
	var result T

	valueBytes, err := json.Marshal(payload)
	if err != nil {
		return result, errors.Internal("marshaling any payload", errors.WithCause(err), errors.WithID("webhook.manager.unmarshal_webhook_value"))
	}

	if err = json.Unmarshal(valueBytes, &result); err != nil {
		return result, errors.Internal("unmarshaling webhook value", errors.WithCause(err), errors.WithID("webhook.manager.unmarshal_webhook_value"))
	}
	return result, nil
}

func (webhookManager *WebhookManager) Type() string { return "whatsapp" }

func (webhookManager *WebhookManager) HandleWebhook(ctx context.Context, payload []byte) error {
	log := webhookManager.logger.With("component", "whatsapp_webhook_manager", "operation", "handle_webhook")

	var webhookPayload WhatsappApiNotificationPayloadSchemaType
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		log.Error("unmarshaling payload data", "error", err, "payload", string(payload))
		return errors.InvalidArgument("unmarshaling paylaod data", errors.WithCause(err), errors.WithID("webhook.manager.handle_webhook"))
	}

	for _, entry := range webhookPayload.Entry {
		for _, change := range entry.Changes {
			switch change.Field {
			case WebhookFieldEnumMessages:
				messageValue, err := unmarshallWebhookValue[MessagesValue](change.Value)
				if err != nil {
					log.Error("unmarshaling webhook value for messages", "error", err)
					return err
				}

				senderName := ""
				if len(messageValue.Contacts) > 0 {
					senderName = messageValue.Contacts[0].Profile.Name
				}

				err = webhookManager.handleMessagesSubscriptionEvents(ctx, handleMessagesSubscriptionEvents{
					Messages:          messageValue.Messages,
					Statuses:          messageValue.Statuses,
					BusinessAccountID: entry.ID,
					SenderName:        senderName,
					PhoneNumber: events.BusinessPhoneNumber{
						DisplayNumber: messageValue.Metadata.DisplayPhoneNumber,
						ID:            messageValue.Metadata.PhoneNumberID,
					},
				})

				if err != nil {
					log.Error("handling messages subscription events", "error", err)
					return errors.Internal("handling messages subscription events", errors.WithCause(err), errors.WithID("webhook.manager.handle_webhook"))
				}
			}
		}
	}

	return nil
}

type handleMessagesSubscriptionEvents struct {
	Messages []Message `json:"messages"`
	Statuses []Status  `json:"statuses"`

	//  business account id to which this event has been sent to
	BusinessAccountID string `json:"business_account_id"`
	SenderName        string `json:"sender_name"`

	//  this is the phone number to which this event has bee sent to
	PhoneNumber events.BusinessPhoneNumber `json:"phone_number"`
}

func (webhookManager *WebhookManager) handleMessagesSubscriptionEvents(ctx context.Context, payload handleMessagesSubscriptionEvents) error {
	for _, message := range payload.Messages {
		repliedTo := message.Context.Id
		baseMessageEvent := events.BaseMessageEvent{
			BusinessAccountID: payload.BusinessAccountID,
			Requester:         webhookManager.RequestClient,
			MessageID:         message.Id,
			From:              message.From,
			SenderName:        payload.SenderName,
			Context:           events.MessageContext{RepliedToMessageID: repliedTo},
			Timestamp:         message.Timestamp,
			IsForwarder:       message.Context.Forwarded,
			PhoneNumber:       payload.PhoneNumber,
		}

		switch message.Type {
		case NotificationMessageTypeText:
			{
				err := webhookManager.coreIntegrationHandler.HandleTextMessage(
					ctx, events.NewTextMessageEven(baseMessageEvent, message.Text.Body),
				)

				if err != nil {
					return err
				}
			}
		case NotificationMessageTypeDocument:
			{
				documentMessage, err := components.NewDocumentMessage(components.DocumentMessageConfigs{
					ID:       message.Document.Id,
					Link:     message.Document.Link,
					Caption:  &message.Document.Caption,
					FileName: message.Document.Filename,
				})

				if err != nil {
					return err
				}

				err = webhookManager.coreIntegrationHandler.HandleDocumentMessage(ctx, events.NewDocumentMessageEvent(
					baseMessageEvent, *documentMessage, message.Document.Id, message.Document.SHA256, message.Document.MIMEType,
				))

				if err != nil {
					return err
				}
			}
		case NotificationMessageTypeImage:
			imageMessage, err := components.NewImageMessage(components.ImageMessageConfigs{
				ID:      message.Image.Id,
				Link:    message.Image.Url,
				Caption: &message.Image.Caption,
			})

			if err != nil {
				return err
			}

			err = webhookManager.coreIntegrationHandler.HandleImageMessage(ctx, events.NewImageMessageEvent(
				baseMessageEvent, *imageMessage, message.Image.Id, message.Image.SHA256, message.Image.MIMEType,
			))

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (webhookManager *WebhookManager) Verify(ctx context.Context, query url.Values) (string, error) {
	var (
		hubVerificationToken = query.Get("hub.verify_token")
		hubChallenge         = query.Get("hub.challenge")
		hubMode              = query.Get("hub.mode")
	)

	return hubChallenge, nil
	if hubMode == "subscribe" && hubVerificationToken == webhookManager.secret {
		return hubChallenge, nil
	}

	return "", errors.InvalidArgument("token does not match provided secret", errors.WithID("webhook.manager.verify"))
}
