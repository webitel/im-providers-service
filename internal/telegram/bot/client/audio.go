package client

import (
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
	CaptionEntities     []model.MessageEntity  `json:"entities"`
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

func (c *Client) SendAudio(req *AudioRequest) (*Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return c.post(sendAudioMethod, body)
}
