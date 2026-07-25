package viber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

// APIBaseURL is the Viber REST bot API root.
// https://developers.viber.com/docs/api/rest-bot-api/#viber-rest-api
const APIBaseURL = "https://chatapi.viber.com/pa"

// authHeader carries the bot's auth token on every request.
const authHeader = "X-Viber-Auth-Token"

// Viber response status codes.
// https://developers.viber.com/docs/api/rest-bot-api/#error-codes
const (
	statusOK                    = 0
	statusInvalidAuthToken      = 2
	statusReceiverNotRegistered = 5
	statusReceiverNotSubscribed = 6
	statusReceiverBlocked       = 7
)

// minAPIVersionKeyboards is required for modern keyboard actions (location-picker, share-phone).
// https://developers.viber.com/docs/api/rest-bot-api/#keyboards
const minAPIVersionKeyboards = 3

// apiError carries an unexpected non-zero Viber status.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("viber: api status %d: %s", e.Status, e.Message)
}

type client struct {
	http    *http.Client
	logger  *slog.Logger
	baseURL string
}

func newClient(l *slog.Logger) *client {
	return &client{
		http:    &http.Client{Timeout: 15 * time.Second},
		logger:  l.With("component", "viber.api"),
		baseURL: APIBaseURL,
	}
}

// statusEnvelope is embedded by every Viber response so post can inspect the status.
type statusEnvelope struct {
	Status        int    `json:"status"`
	StatusMessage string `json:"status_message"`
}

// post sends a JSON body to path authenticated with token. Viber returns HTTP 200
// even on logical failure, so the response body status field is authoritative.
// When out is non-nil the raw body is additionally decoded into it.
func (c *client) post(ctx context.Context, token, path string, reqBody, out any) error {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("viber: marshal %s: %w", path, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(authHeader, token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("viber %s: http %d: %s", path, resp.StatusCode, body)
	}

	var env statusEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("viber %s: decode status: %w", path, err)
	}
	if env.Status != statusOK {
		return statusToError(env.Status, env.StatusMessage)
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}

func statusToError(code int, msg string) error {
	switch code {
	case statusInvalidAuthToken:
		return vibmodel.ErrTokenInvalid
	case statusReceiverNotRegistered, statusReceiverNotSubscribed, statusReceiverBlocked:
		return vibmodel.ErrReceiverNotSubscribed
	default:
		return &apiError{Status: code, Message: msg}
	}
}

type accountInfoResponse struct {
	statusEnvelope
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
	Icon string `json:"icon"`
}

// GetAccountInfo validates the token and returns the bot's public account details.
func (c *client) GetAccountInfo(ctx context.Context, token string) (*vibmodel.AccountInfo, error) {
	var out accountInfoResponse
	if err := c.post(ctx, token, "/get_account_info", struct{}{}, &out); err != nil {
		return nil, err
	}
	return &vibmodel.AccountInfo{ID: out.ID, Name: out.Name, URI: out.URI, Avatar: out.Icon}, nil
}

type setWebhookRequest struct {
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types,omitempty"`
	SendName   bool     `json:"send_name"`
	SendPhoto  bool     `json:"send_photo"`
}

// SetWebhook registers the callback URL. send_name/send_photo make Viber include the
// sender profile inline on every message event, so no profile round-trip is needed.
func (c *client) SetWebhook(ctx context.Context, token, url string) error {
	return c.post(ctx, token, "/set_webhook", setWebhookRequest{
		URL:        url,
		EventTypes: []string{"message", "subscribed", "unsubscribed", "conversation_started", "delivered", "seen", "failed"},
		SendName:   true,
		SendPhoto:  true,
	}, nil)
}

// RemoveWebhook clears the registered callback URL (Viber removes it when url is empty).
func (c *client) RemoveWebhook(ctx context.Context, token string) error {
	return c.post(ctx, token, "/set_webhook", setWebhookRequest{URL: ""}, nil)
}

type sender struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type msgLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type msgContact struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phone_number"`
}

type sendMessageRequest struct {
	Receiver      string       `json:"receiver"`
	MinAPIVersion int          `json:"min_api_version,omitempty"`
	Sender        sender       `json:"sender"`
	Type          string       `json:"type"`
	Text          string       `json:"text,omitempty"`
	Media         string       `json:"media,omitempty"`
	Size          int64        `json:"size,omitempty"`
	FileName      string       `json:"file_name,omitempty"`
	Location      *msgLocation `json:"location,omitempty"`
	Contact       *msgContact  `json:"contact,omitempty"`
	Keyboard      *keyboard    `json:"keyboard,omitempty"`
}

type sendResponse struct {
	statusEnvelope
	MessageToken json.Number `json:"message_token"`
}

func (c *client) send(ctx context.Context, token string, msg sendMessageRequest) (*sharedmodel.MessageResponse, error) {
	if msg.Keyboard != nil {
		msg.MinAPIVersion = minAPIVersionKeyboards
	}
	var out sendResponse
	if err := c.post(ctx, token, "/send_message", msg, &out); err != nil {
		return nil, err
	}
	return &sharedmodel.MessageResponse{ID: out.MessageToken.String()}, nil
}

// SendText delivers a text message, optionally with an attached keyboard.
func (c *client) SendText(ctx context.Context, token string, s sender, receiver, text string, kb *keyboard) (*sharedmodel.MessageResponse, error) {
	return c.send(ctx, token, sendMessageRequest{
		Receiver: receiver, Sender: s, Type: "text", Text: text, Keyboard: kb,
	})
}

// SendPicture delivers an image by URL with an optional caption. Viber fetches the media server-side.
func (c *client) SendPicture(ctx context.Context, token string, s sender, receiver, mediaURL, caption string) (*sharedmodel.MessageResponse, error) {
	return c.send(ctx, token, sendMessageRequest{
		Receiver: receiver, Sender: s, Type: "picture", Media: mediaURL, Text: caption,
	})
}

// SendFile delivers a document by URL. size is required by Viber for file messages.
func (c *client) SendFile(ctx context.Context, token string, s sender, receiver, mediaURL, fileName string, size int64) (*sharedmodel.MessageResponse, error) {
	return c.send(ctx, token, sendMessageRequest{
		Receiver: receiver, Sender: s, Type: "file", Media: mediaURL, FileName: fileName, Size: size,
	})
}

// SendLocation delivers a geographic point.
func (c *client) SendLocation(ctx context.Context, token string, s sender, receiver string, lat, lon float64) (*sharedmodel.MessageResponse, error) {
	return c.send(ctx, token, sendMessageRequest{
		Receiver: receiver, Sender: s, Type: "location", Location: &msgLocation{Lat: lat, Lon: lon},
	})
}

// SendContact delivers a contact card.
func (c *client) SendContact(ctx context.Context, token string, s sender, receiver, name, phone string) (*sharedmodel.MessageResponse, error) {
	return c.send(ctx, token, sendMessageRequest{
		Receiver: receiver, Sender: s, Type: "contact", Contact: &msgContact{Name: name, PhoneNumber: phone},
	})
}
