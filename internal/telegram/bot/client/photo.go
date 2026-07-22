package client

import (
	"encoding/json"

	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

type PhotoRequest struct {
	ChatID                int64                  `json:"chat_id"`
	Photo                 string                 `json:"photo"`
	Caption               *string                `json:"caption"`
	ParseMode             *string                `json:"parse_mode"`
	Entities              []model.MessageEntity  `json:"entities"`
	ShowCaptionAboveMedia bool                   `json:"show_caption_above_media"`
	HasSpoiler            bool                   `json:"has_spoiler"`
	DisableNotification   bool                   `json:"disable_notification"`
	ProtectContent        bool                   `json:"protect_content"`
	ReplyParameters       *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup           OutgoingKeyboarder     `json:"reply_markup"`
}

func (c *Client) SendPhoto(req *PhotoRequest) (*Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return c.post("sendPhoto", body)
}
