package client

import (
	"encoding/json"

	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

const (
	DocumentMethod = "sendDocument"
)

type DocumentRequest struct {
	ChatID              int64                  `json:"chat_id"`
	Document            string                 `json:"document"`
	Thumbnail           *string                `json:"thumbnail"`
	Caption             *string                `json:"caption"`
	CaptionEntities     []model.MessageEntity  `json:"entities"`
	ParseMode           *string                `json:"parse_mode"`
	DisableNotification bool                   `json:"disable_notification"`
	ProtectContent      bool                   `json:"protect_content"`
	ReplyParameters     *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup         OutgoingKeyboarder     `json:"reply_markup"`
}

func (c *Client) SendDocument(req *DocumentRequest) (*Response, error) {
	url := c.registry.sendDocument

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return c.post(url, body)
}
