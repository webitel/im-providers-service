package model

type StickerType string

const (
	StickerTypeRegular     StickerType = "regular"
	StickerTypeMask        StickerType = "mask"
	StickerTypeCustomEmoji StickerType = "custom_emoji"
)

// https://core.telegram.org/bots/api#sticker
type Sticker struct {
	FileID          string      `json:"file_id"`
	FileUniqueID    string      `json:"file_unique_id"`
	Type            StickerType `json:"type"`
	Width           int64       `json:"width"`
	Height          int64       `json:"height"`
	IsAnimated      bool        `json:"is_animated"`
	IsVideo         bool        `json:"is_video"`
	Thumbnail       *PhotoSize  `json:"thumbnail,omitempty"`
	Emoji           *string     `json:"emoji,omitempty"`
	SetName         *string     `json:"set_name,omitempty"`
	CustomEmojiID   *string     `json:"custom_emoji_id,omitempty"`
	NeedsRepainting *bool       `json:"needs_repainting,omitempty"`
	FileSize        *int64      `json:"file_size,omitempty"`
}