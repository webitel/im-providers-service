package model

// https://core.telegram.org/bots/api#callbackquery
// This object represents an incoming callback query from a callback button in an inline keyboard.
//
//	If the button that originated the query was attached
//	to a message sent by the bot, the field message will be present.
//	If the button was attached to a message sent via the bot (in inline mode),
//	the field inline_message_id will be present.
//	Exactly one of the fields data or game_short_name will be present.
type CallbackQuery struct {
	ID   string `json:"id"`
	From *User  `json:"from"`
	// telegram describes this field as MaybeInaccessibleMessage,
	// but because our bot can't be a member of the group, and the message with buttons
	// always comes from the bot, we can safely treat it as a regular Message.
	Message         *Message `json:"message,omitempty"`
	InlineMessageID *string  `json:"inline_message_id,omitempty"`
	ChatInstance    string   `json:"chat_instance"`
	Data            *string  `json:"data,omitempty"`
}
