package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// https://core.telegram.org/bots/api#making-requests
type apiResponse struct {
	Ok          bool                     `json:"ok"`
	Result      json.RawMessage          `json:"result,omitempty"`
	ErrorCode   int                      `json:"error_code,omitempty"`
	Description string                   `json:"description,omitempty"`
	Parameters  *ErrorResponseParameters `json:"parameters,omitempty"`
}

type ErrorResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id"`
	RetryAfter      int64 `json:"retry_after"`
}

// APIError is returned when Telegram responds with "ok": false.
// https://core.telegram.org/bots/api#making-requests
type APIError struct {
	Method      string
	StatusCode  int
	ErrorCode   int
	Description string
	Parameters  *ErrorResponseParameters
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram: %s failed (http %d, code %d): %s", e.Method, e.StatusCode, e.ErrorCode, e.Description)
}

const (
	baseTelegramURL = "https://api.telegram.org/bot%s/%s"
)

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

type Client struct {
	httpClient *http.Client
}

// callMethod invokes a Telegram Bot API method and, on success, unmarshals the
// "result" field of the response envelope into out (when non-nil).
// https://core.telegram.org/bots/api#making-requests
func (c *Client) callMethod(ctx context.Context, methodName, token string, body []byte, out any) error {
	url := ConstructURL(token, methodName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("telegram: decode %s response: %w", methodName, err)
	}

	if !envelope.Ok {
		return &APIError{
			Method:      methodName,
			StatusCode:  resp.StatusCode,
			ErrorCode:   envelope.ErrorCode,
			Description: envelope.Description,
			Parameters:  envelope.Parameters,
		}
	}

	if out != nil && len(envelope.Result) > 0 {
		if err := json.Unmarshal(envelope.Result, out); err != nil {
			return fmt.Errorf("telegram: unmarshal %s result: %w", methodName, err)
		}
	}

	return nil
}

type OutgoingKeyboarder interface {
	MarshalJSON() ([]byte, error)
	OutgoingKeyboard()
}

func ConstructURL(token, method string) string {
	return fmt.Sprintf(baseTelegramURL, token, method)
}
