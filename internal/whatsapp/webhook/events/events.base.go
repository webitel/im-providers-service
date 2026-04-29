package events

import "github.com/webitel/im-providers-service/internal/whatsapp/client"

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

type BaseMediaMessageEvent struct {
	BaseMessageEvent `json:",inline"`
	MediaId          string `json:"media_id"`
	MimeType         string `json:"mime_type"`
	Sha256           string `json:"sha256"`
}
