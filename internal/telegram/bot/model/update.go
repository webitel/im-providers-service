package model

type WebhookUpdate struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	EditedMessage *Message       `json:"edited_message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
	Poll          *Poll          `json:"poll,omitempty"`
	PollAnswer    *PollAnswer    `json:"poll_answer,omitempty"`
}
