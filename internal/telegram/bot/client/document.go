package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

const (
	sendDocumentMethod  = "sendDocument"
	getFileMethod       = "getFile"
	documentURLTemplate = "https://api.telegram.org/file/bot%s/%s"
)

type DocumentRequest struct {
	ChatID              int64                  `json:"chat_id"`
	Document            string                 `json:"document"`
	Thumbnail           *string                `json:"thumbnail"`
	Caption             *string                `json:"caption"`
	CaptionEntities     []model.MessageEntity  `json:"caption_entities"`
	ParseMode           *string                `json:"parse_mode"`
	DisableNotification bool                   `json:"disable_notification"`
	ProtectContent      bool                   `json:"protect_content"`
	ReplyParameters     *model.ReplyParameters `json:"reply_parameters"`
	ReplyMarkup         OutgoingKeyboarder     `json:"reply_markup"`
}

type GetDocumentRequest struct {
	FileID string `json:"file_id"`
}

// https://core.telegram.org/bots/api#file
type File struct {
	FileID       string  `json:"file_id"`
	FileUniqueID string  `json:"file_unique_id"`
	FileSize     *int64  `json:"file_size,omitempty"`
	FilePath     *string `json:"file_path,omitempty"`
}

func (c *Client) SendDocument(ctx context.Context, token string, req *DocumentRequest) (*model.Message, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var result model.Message
	if err := c.callMethod(ctx, sendDocumentMethod, token, body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateFileURL resolves a file_id to a downloadable URL via getFile.
// https://core.telegram.org/bots/api#getfile
func (c *Client) GenerateFileURL(ctx context.Context, token string, fileID string) (*url.URL, error) {
	body, err := json.Marshal(GetDocumentRequest{FileID: fileID})
	if err != nil {
		return nil, err
	}

	var file File
	if err := c.callMethod(ctx, getFileMethod, token, body, &file); err != nil {
		return nil, err
	}

	if file.FilePath == nil {
		return nil, errors.Internal("file path required to build file url is not returned from telegram")
	}

	return url.Parse(fmt.Sprintf(documentURLTemplate, token, *file.FilePath))
}
