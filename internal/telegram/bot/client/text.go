package client

import (
	"encoding/json"

	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

const (
	sendTextMethod = "sendMessage"
)

type TextRequest struct {
	ChatID                int64                  `json:"chat_id"`
	Text                  string                 `json:"text"`
	DirectMessagesTopicID int64                  `json:"direct_messages_topic_id,omitempty"`
	ParseMode             string                 `json:"parse_mode,omitempty"`
	Entities              []model.MessageEntity  `json:"entities,omitempty"`
	ProtectContent        bool                   `json:"protect_content,omitempty"`
	ReplyParameters       *model.ReplyParameters `json:"reply_parameters,omitempty"`
	ReplyMarkup           OutgoingKeyboarder     `json:"reply_markup,omitempty"`
}

func (c *Client) SendText(req *TextRequest) (*Response, error) {
	url := c.registry.sendText

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.post(url, reqBody)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
