package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
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
	ReplyParameters     *model.ReplyParameters `json:"reply_parameters,omitempty"`
	ReplyMarkup         OutgoingKeyboarder     `json:"reply_markup,omitempty"`
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

// DownloadFile resolves file_id via getFile and opens the file body. The
// download URL embeds the bot token, so it never leaves this client: callers
// get the body only, and transport errors are redacted.
// https://core.telegram.org/bots/api#getfile
func (c *Client) DownloadFile(ctx context.Context, token, fileID string) (io.ReadCloser, *File, error) {
	body, err := json.Marshal(GetDocumentRequest{FileID: fileID})
	if err != nil {
		return nil, nil, err
	}

	var file File
	if err := c.callMethod(ctx, getFileMethod, token, body, &file); err != nil {
		return nil, nil, err
	}

	if file.FilePath == nil {
		return nil, nil, errors.Internal("file path required to download file is not returned from telegram")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(documentURLTemplate, token, *file.FilePath), nil)
	if err != nil {
		return nil, nil, redactToken(err, token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, redactToken(err, token)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()

		return nil, nil, fmt.Errorf("telegram: download file: status %s", resp.Status)
	}

	return resp.Body, &file, nil
}
