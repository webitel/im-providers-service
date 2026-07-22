package model

// https://core.telegram.org/bots/api#replyparameters
type ReplyParameters struct {
	MessageID                *int64          `json:"message_id,omitempty"`
	ChatID                   string          `json:"chat_id,omitempty"`
	EphemeralMessageID       *int64          `json:"ephemeral_message_id,omitempty"`
	AllowSendingWithoutReply *bool           `json:"allow_sending_without_reply,omitempty"`
	Quote                    *string         `json:"quote,omitempty"`
	QuoteParseMode           *string         `json:"quote_parse_mode,omitempty"`
	QuoteEntities            []MessageEntity `json:"quote_entities,omitempty"`
	QuotePosition            *int64          `json:"quote_position,omitempty"`
	ChecklistTaskID          *int64          `json:"checklist_task_id,omitempty"`
	PollOptionID             *string         `json:"poll_option_id,omitempty"`
}
