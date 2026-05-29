package events

import (
	"context"
	"net/http"

	"github.com/webitel/im-providers-service/internal/whatsapp/client"
	"github.com/webitel/im-providers-service/internal/whatsapp/messaging/components"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type BaseMessage interface {
	ToJSON(configs components.ApiCompatibleJsonConverterConfigs) (string, error)
}

type BusinessPhoneNumber struct {
	DisplayNumber string `json:"display_number"`
	ID            string `json:"id"`
}

type MessageContext struct {
	RepliedToMessageID string `json:"replied_to_message_id"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type BaseMessageEvent struct {
	BusinessAccountID string               `json:"business_account_id"`
	Requester         client.RequestClient `json:"requester"`
	MessageID         string               `json:"message_id"`
	From              string               `json:"from"`
	SenderName        string               `json:"sender_name"`
	Context           MessageContext       `json:"context"`
	Timestamp         string               `json:"timestamp"`
	IsForwarder       bool                 `json:"is_forwarder"`
	PhoneNumber       BusinessPhoneNumber  `json:"phone_number"`
	Metadata          Metadata             `json:"metadata"`
}

func (baseMessageEvent *BaseMessageEvent) GetEventType() string { return "message" }

func (baseMessageEvent *BaseMessageEvent) Reply(ctx context.Context, message BaseMessage) (string, error) {
	body, err := message.ToJSON(components.ApiCompatibleJsonConverterConfigs{
		ReplyToMessage:     baseMessageEvent.MessageID,
		SendingPhoneNumber: baseMessageEvent.SenderName,
	})

	if err != nil {
		return "", errors.Internal(
			"converting message to json",
			errors.WithCause(err),
			errors.WithID("events.base.reply"),
			errors.WithValue("from", baseMessageEvent.From),
			errors.WithValue("message_id", baseMessageEvent.MessageID),
		)
	}

	apiRequest := baseMessageEvent.Requester.NewApiRequest(
		baseMessageEvent.PhoneNumber.ID+"/messages",
		http.MethodPost,
	)
	apiRequest.SetBody(body)

	response, err := apiRequest.ExecuteWithContext(ctx)
	if err != nil {
		return "", errors.New(
			"executing wapi reply request",
			errors.WithCause(err),
			errors.WithID("events.base.reply"),
			errors.WithValue("from", baseMessageEvent.From),
			errors.WithValue("message_id", baseMessageEvent.MessageID),
		)
	}

	return response, nil
}

type BaseMediaMessageEvent struct {
	BaseMessageEvent `json:",inline"`

	MediaID  string `json:"media_id"`
	MimeType string `json:"mime_type"`
	Sha256   string `json:"sha256"`
}
