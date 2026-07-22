package model

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

func (i *InlineKeyboardMarkup) OutgoingKeyboard() {

}

func (i *InlineKeyboardMarkup) ToJSON() ([]byte, error) {
	// TODO
	return nil, nil
}

// https://core.telegram.org/bots/api#replykeyboardmarkup
type ReplyKeyboardMarkup struct {
	Keyboard              [][]KeyboardButton `json:"keyboard"`
	IsPersistent          *bool              `json:"is_persistent,omitempty"`
	ResizeKeyboard        *bool              `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard       *bool              `json:"one_time_keyboard,omitempty"`
	InputFieldPlaceholder *string            `json:"input_field_placeholder,omitempty"`
	Selective             *bool              `json:"selective,omitempty"`
}

func (r *ReplyKeyboardMarkup) OutgoingKeyboard() {

}

func (r *ReplyKeyboardMarkup) ToJSON() ([]byte, error) {
	// TODO
	return nil, nil
}

// https://core.telegram.org/bots/api#replykeyboardremove
type ReplyKeyboardRemove struct {
	RemoveKeyboard *bool `json:"remove_keyboard"`
	Selective      *bool `json:"selective,omitempty"`
}

func (r *ReplyKeyboardRemove) OutgoingKeyboard() {

}

func (r *ReplyKeyboardRemove) ToJSON() ([]byte, error) {
	// TODO
	return nil, nil
}

// https://core.telegram.org/bots/api#forcereply
type ForceReply struct {
	ForceReply            bool    `json:"force_reply"`
	InputFieldPlaceholder *string `json:"input_field_placeholder,omitempty"`
	Selective             *bool   `json:"selective,omitempty"`
}

func (f *ForceReply) OutgoingKeyboard() {

}

func (f *ForceReply) ToJSON() ([]byte, error) {
	// TODO
	return nil, nil
}

// https://core.telegram.org/bots/api#keyboardbutton
type KeyboardButton struct {
	Text              string                           `json:"text"`
	IconCustomEmojiID *string                          `json:"icon_custom_emoji_id,omitempty"`
	Style             *string                          `json:"style,omitempty"`
	RequestUsers      *KeyboardButtonRequestUsers      `json:"request_users,omitempty"`
	RequestChat       *KeyboardButtonRequestChat       `json:"request_chat,omitempty"`
	RequestManagedBot *KeyboardButtonRequestManagedBot `json:"request_managed_bot,omitempty"`
	RequestContact    *bool                            `json:"request_contact,omitempty"`
	RequestLocation   *bool                            `json:"request_location,omitempty"`
	RequestPoll       *KeyboardButtonPollType          `json:"request_poll,omitempty"`
	WebApp            *WebAppInfo                      `json:"web_app,omitempty"`
}

// TODO: fill in fields per https://core.telegram.org/bots/api as each is needed.
type (
	KeyboardButtonRequestUsers      struct{}
	KeyboardButtonRequestChat       struct{}
	KeyboardButtonRequestManagedBot struct{}
	KeyboardButtonPollType          struct{}
)
