package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Response struct {
	Ok               bool                     `json:"ok"`
	SuccessResult    *string                  `json:"result,omitempty"`
	ErrorDescription *string                  `json:"description,omitempty"`
	ErrorParameters  *ErrorResponseParameters `json:"parameters,omitempty"`
}

type ErrorResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id"`
	RetryAfter      int64 `json:"retry_after"`
}

const (
	baseTelegramURL = "https://api.telegram.org/bot%s/%s"
)

func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{},
		registry: &methodRegistry{
			sendText:     ConstructURL(token, sendTextMethod),
			sendDocument: ConstructURL(token, DocumentMethod),
		},
	}
}

type Client struct {
	token      string
	httpClient *http.Client

	registry *methodRegistry
}

type methodRegistry struct {
	sendText      string
	sendPhoto     string
	sendVideo     string
	sendDocument  string
	sendVoice     string
	sendSticker   string
	sendAnimation string
	sendAudio     string
}

func (c *Client) post(url string, body []byte) (*Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}

type OutgoingKeyboarder interface {
	MarshalJSON() ([]byte, error)
	OutgoingKeyboard()
}

func ConstructURL(token string, method string) string {
	return fmt.Sprintf(baseTelegramURL, token, method)
}
