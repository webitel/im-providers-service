package model

import "strings"

// https://core.telegram.org/bots/api#messageorigin
type MessageOriginType string

const (
	MessageOriginTypeUser       MessageOriginType = "user"
	MessageOriginTypeHiddenUser MessageOriginType = "hidden_user"
	MessageOriginTypeChat       MessageOriginType = "chat"
	MessageOriginTypeChannel    MessageOriginType = "channel"
)

// https://core.telegram.org/bots/api#messageorigin
type MessageOrigin struct {
	Type            MessageOriginType `json:"type"`
	Date            int64             `json:"date"`
	SenderUser      *User             `json:"sender_user,omitempty"`
	SenderUserName  *string           `json:"sender_user_name,omitempty"`
	SenderChat      *Chat             `json:"sender_chat,omitempty"`
	Chat            *Chat             `json:"chat,omitempty"`
	MessageID       *int64            `json:"message_id,omitempty"`
	AuthorSignature *string           `json:"author_signature,omitempty"`
}

func (o *MessageOrigin) SenderName() string {
	if o == nil {
		return ""
	}

	switch o.Type {
	case MessageOriginTypeUser:
		if o.SenderUser == nil {
			return ""
		}

		name := o.SenderUser.FirstName
		if o.SenderUser.LastName != nil && *o.SenderUser.LastName != "" {
			name = strings.TrimSpace(name + " " + *o.SenderUser.LastName)
		}

		if name == "" && o.SenderUser.Username != nil {
			name = *o.SenderUser.Username
		}

		return name

	case MessageOriginTypeHiddenUser:
		if o.SenderUserName != nil {
			return *o.SenderUserName
		}

	case MessageOriginTypeChat:
		return chatOrSignatureName(o.SenderChat, o.AuthorSignature)

	case MessageOriginTypeChannel:
		return chatOrSignatureName(o.Chat, o.AuthorSignature)
	}

	return ""
}

func (o *MessageOrigin) OriginalSentAtMillis() int64 {
	if o == nil || o.Date <= 0 {
		return 0
	}

	return o.Date * 1000
}

func chatOrSignatureName(chat *Chat, signature *string) string {
	if signature != nil && *signature != "" {
		return *signature
	}

	if chat == nil {
		return ""
	}

	if chat.Title != nil && *chat.Title != "" {
		return *chat.Title
	}

	if chat.Username != nil {
		return *chat.Username
	}

	return ""
}
