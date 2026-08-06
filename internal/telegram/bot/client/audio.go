package client

import (
	"context"
	"encoding/json"

	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

const (
	sendAudioMethod = "sendAudio"
)

type AudioRequest struct {
	ChatID              int64                  `json:"chat_id"`
	Audio               string                 `json:"audio"`
	Caption             *string                `json:"caption"`
	CaptionEntities     []model.MessageEntity  `json:"caption_entities"`
	ParseMode           *string                `json:"parse_mode"`
	Duration            *int64                 `json:"duration"` // In seconds
	Title               *string                `json:"title"`
	Performer           *string                `json:"performer"`
	Thumbnail           *string                `json:"thumbnail"`
	DisableNotification bool                   `json:"disable_notification"`
	ProtectContent      bool                   `json:"protect_content"`
	ReplyParameters     *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup         OutgoingKeyboarder     `json:"reply_markup"`
}

func (c *Client) SendAudio(ctx context.Context, token string, req *AudioRequest) (*model.Message, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var message model.Message
	if err := c.callMethod(ctx, sendAudioMethod, token, body, &message); err != nil {
		return nil, err
	}

	return &message, nil
}
