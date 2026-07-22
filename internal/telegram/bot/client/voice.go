package client

import "github.com/webitel/im-providers-service/internal/telegram/bot/model"

type VoiceRequest struct {
	ChatID              int64                  `json:"chat_id"`
	Voice               string                 `json:"voice"`
	Caption             *string                `json:"caption"`
	CaptionEntities     []model.MessageEntity  `json:"entities"`
	ParseMode           *string                `json:"parse_mode"`
	Duration            *int64                 `json:"duration"` // In seconds
	DisableNotification bool                   `json:"disable_notification"`
	ProtectContent      bool                   `json:"protect_content"`
	ReplyParameters     *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup         OutgoingKeyboarder
}

func (c *Client) SendVoice(req *VoiceRequest) (*Response, error) {
	// TODO:

	return nil, nil
}
