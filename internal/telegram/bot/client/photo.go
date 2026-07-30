package client

import (
	"context"
	"encoding/json"

	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

const (
	sendPhotoMethod = "sendPhoto"
)

type PhotoRequest struct {
	ChatID                int64                  `json:"chat_id"`
	Photo                 string                 `json:"photo"`
	Caption               *string                `json:"caption"`
	ParseMode             *string                `json:"parse_mode"`
	Entities              []model.MessageEntity  `json:"caption_entities"`
	ShowCaptionAboveMedia bool                   `json:"show_caption_above_media"`
	HasSpoiler            bool                   `json:"has_spoiler"`
	DisableNotification   bool                   `json:"disable_notification"`
	ProtectContent        bool                   `json:"protect_content"`
	ReplyParameters       *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup           OutgoingKeyboarder     `json:"reply_markup"`
}

func (c *Client) SendPhoto(ctx context.Context, token string, req *PhotoRequest) (*model.Message, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var message model.Message
	if err := c.callMethod(ctx, sendPhotoMethod, token, body, &message); err != nil {
		return nil, err
	}

	return &message, nil
}
