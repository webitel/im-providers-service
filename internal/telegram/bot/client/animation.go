package client

import "github.com/webitel/im-providers-service/internal/telegram/bot/model"

type AnimationRequest struct {
	ChatID                int64                  `json:"chat_id"`
	Animation             string                 `json:"animation"`
	Duration              *int64                 `json:"duration"`
	Width                 *int64                 `json:"width"`
	Height                *int64                 `json:"height"`
	Thumbnail             *string                `json:"thumbnail"`
	Caption               *string                `json:"caption"`
	CaptionEntities       []model.MessageEntity  `json:"entities"`
	ParseMode             *string                `json:"parse_mode"`
	ShowCaptionAboveMedia bool                   `json:"show_caption_above_media"`
	HasSpoiler            bool                   `json:"has_spoiler"`
	DisableNotification   bool                   `json:"disable_notification"`
	ProtectContent        bool                   `json:"protect_content"`
	ReplyParameters       *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup           OutgoingKeyboarder
}

func (c *Client) SendAnimation(req *AnimationRequest) (*Response, error) {
	// TODO:

	return nil, nil
}
