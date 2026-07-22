package client

import "github.com/webitel/im-providers-service/internal/telegram/bot/model"

type VideoRequest struct {
	ChatID                int64                  `json:"chat_id"`
	Video                 string                 `json:"video"`
	Caption               *string                `json:"caption"`
	CaptionEntities       []model.MessageEntity  `json:"entities"`
	ParseMode             *string                `json:"parse_mode"`
	Duration              *int64                 `json:"duration"` // In seconds
	Width                 *int64                 `json:"width"`
	Height                *int64                 `json:"height"`
	Thumbnail             *string                `json:"thumbnail"`
	Cover                 *string                `json:"cover"`
	StartTimestamp        *int64                 `json:"start_timestamp"`
	ShowCaptionAboveMedia bool                   `json:"show_caption_above_media"`
	HasSpoiler            bool                   `json:"has_spoiler"`
	SupportsStreaming     bool                   `json:"supports_streaming"`
	DisableNotification   bool                   `json:"disable_notification"`
	ProtectContent        bool                   `json:"protect_content"`
	ReplyParameters       *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup           OutgoingKeyboarder
}

func (c *Client) SendVideo(req *VideoRequest) (*Response, error) {
	// TODO:

	return nil, nil
}
