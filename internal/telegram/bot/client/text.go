package client

import (
	"context"
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

func (c *Client) SendText(ctx context.Context, token string, req *TextRequest) (*model.Message, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var message model.Message
	if err := c.callMethod(ctx, sendTextMethod, token, reqBody, &message); err != nil {
		return nil, err
	}

	return &message, nil
}
