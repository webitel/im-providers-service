package model

type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSupergroup ChatType = "supergroup"
	ChatTypeChannel    ChatType = "channel"
)

// https://core.telegram.org/bots/api#chat
type Chat struct {
	ID               int64    `json:"id"`
	Type             ChatType `json:"type"`
	Title            *string  `json:"title,omitempty"`
	Username         *string  `json:"username,omitempty"`
	FirstName        *string  `json:"first_name,omitempty"`
	LastName         *string  `json:"last_name,omitempty"`
	IsForum          *bool    `json:"is_forum,omitempty"`
	IsDirectMessages *bool    `json:"is_direct_messages,omitempty"`
}
