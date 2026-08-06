package model

// https://core.telegram.org/bots/api#message
type Message struct {
	MessageID             int64                `json:"message_id"`
	MessageThreadID       *int64               `json:"message_thread_id,omitempty"`
	DirectMessagesTopic   *DirectMessagesTopic `json:"direct_messages_topic,omitempty"`
	Text                  *string              `json:"text,omitempty"`
	From                  *User                `json:"from,omitempty"`
	SenderChat            *Chat                `json:"sender_chat,omitempty"`
	SenderBoostCount      *int64               `json:"sender_boost_count,omitempty"`
	SenderBusinessBot     *User                `json:"sender_business_bot,omitempty"`
	SenderTag             *string              `json:"sender_tag,omitempty"`
	ReceiverUser          *User                `json:"receiver_user,omitempty"`
	EphemeralMessageID    *int64               `json:"ephemeral_message_id,omitempty"`
	Date                  int64                `json:"date"`
	GuestQueryID          *string              `json:"guest_query_id,omitempty"`
	BussinessConnectionID *string              `json:"business_connection_id,omitempty"`
	Chat                  *Chat                `json:"chat"`
	ForwardOrigin         *MessageOrigin       `json:"forward_origin,omitempty"`
	ReplyToMessage        *Message             `json:"reply_to_message,omitempty"`
	Quote                 *TextQuote           `json:"quote,omitempty"`
	EditDate              *int64               `json:"edit_date,omitempty"`
	Entities              []MessageEntity      `json:"entities,omitempty"`
	RichMessage           *RichMessage         `json:"rich_message,omitempty"`
	Animation             *Animation           `json:"animation,omitempty"` // GIF
	Audio                 *Audio               `json:"audio,omitempty"`
	Document              *Document            `json:"document,omitempty"`
	Photo                 []PhotoSize          `json:"photo,omitempty"`
	Sticker               *Sticker             `json:"sticker,omitempty"`
	Video                 *Video               `json:"video,omitempty"`
	VideoNote             *VideoNote           `json:"video_note,omitempty"`
	Voice                 *Voice               `json:"voice,omitempty"`
	Contact               *Contact             `json:"contact,omitempty"`
	Poll                  *Poll                `json:"poll,omitempty"`
	Location              *Location            `json:"location,omitempty"`
	PollOptionAdded       *PollOptionAdded     `json:"poll_option_added,omitempty"`
	PollOptionDeleted     *PollOptionDeleted   `json:"poll_option_deleted,omitempty"`
	ReplyMarkup           *ReplyMarkup         `json:"reply_markup,omitempty"`
}

type TextQuote struct {
	Text     string          `json:"text"`
	Position int64           `json:"position"`
	IsManual *bool           `json:"is_manual,omitempty"`
	Entities []MessageEntity `json:"entities,omitempty"`
}

// https://core.telegram.org/bots/api#animation
type Animation struct {
	FileID       string     `json:"file_id"`
	FileUniqueID string     `json:"file_unique_id"`
	Width        int64      `json:"width"`
	Height       int64      `json:"height"`
	Duration     int64      `json:"duration"`
	Thumbnail    *PhotoSize `json:"thumbnail,omitempty"`
	FileName     *string    `json:"file_name,omitempty"`
	MimeType     *string    `json:"mime_type,omitempty"`
	FileSize     *int64     `json:"file_size,omitempty"`
}

// https://core.telegram.org/bots/api#photosize
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int64  `json:"width"`
	Height       int64  `json:"height"`
	FileSize     *int64 `json:"file_size,omitempty"`
}

type RichMessage struct {
	// TODO: new telegram feature, add later
}

type MessageEntity struct {
	// TODO: add fields
}

type DirectMessagesTopic struct {
	TopicID int64 `json:"topic_id"`
	User    *User `json:"user"`
}
