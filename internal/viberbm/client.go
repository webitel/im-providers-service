package viberbm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// apiClient is a thin JSON client over the InfoBip Viber BM API; every call is
// parameterised by the gate's BaseURL and App key, so one instance serves all.
type apiClient struct {
	http   *http.Client
	logger *slog.Logger
}

func newAPIClient(l *slog.Logger) *apiClient {
	return &apiClient{
		http:   &http.Client{Timeout: 15 * time.Second},
		logger: l.With("component", "viber_bm.api"),
	}
}

type envelope struct {
	Messages []outMessage `json:"messages"`
}

type outMessage struct {
	Sender       string `json:"sender"`
	Destinations []dest `json:"destinations"`
	Content      any    `json:"content"`
	// MessageID echoes our internal id for InfoBip dedup and DLR correlation.
	MessageID string `json:"messageId,omitempty"` //nolint:tagliatelle // InfoBip wire format
}

type dest struct {
	To string `json:"to"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type imageContent struct {
	Type     string `json:"type"`
	MediaURL string `json:"mediaUrl"`
	Text     string `json:"text,omitempty"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type fileContent struct {
	Type     string `json:"type"`
	MediaURL string `json:"mediaUrl"`
	FileName string `json:"fileName"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type templateContent struct {
	Type       string            `json:"type"`
	TemplateID string            `json:"templateId"`
	Language   string            `json:"language,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type sendResponse struct {
	BulkID   string        `json:"bulkId"`
	Messages []sentMessage `json:"messages"`
	Error    *requestError `json:"requestError,omitempty"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type sentMessage struct {
	MessageID string     `json:"messageId"`
	Status    sentStatus `json:"status"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type sentStatus struct {
	GroupID     int    `json:"groupId"`
	GroupName   string `json:"groupName"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

//nolint:tagliatelle // InfoBip omni-channel wire format is camelCase
type requestError struct {
	ServiceException struct {
		MessageID string `json:"messageId"`
		Text      string `json:"text"`
	} `json:"serviceException"`
}

type sendResult struct {
	BulkID    string
	MessageID string
	GroupName string
}

func (c *apiClient) SendText(ctx context.Context, base, apiKey, sender, to, text, messageID string) (*sendResult, error) {
	return c.send(ctx, base, apiKey, outMessage{
		Sender:       sender,
		Destinations: []dest{{To: to}},
		Content:      textContent{Type: contentTypeText, Text: text},
		MessageID:    messageID,
	})
}

func (c *apiClient) SendImage(ctx context.Context, base, apiKey, sender, to, imageURL, caption, messageID string) (*sendResult, error) {
	return c.send(ctx, base, apiKey, outMessage{
		Sender:       sender,
		Destinations: []dest{{To: to}},
		Content:      imageContent{Type: contentTypeImage, MediaURL: imageURL, Text: caption},
		MessageID:    messageID,
	})
}

func (c *apiClient) SendFile(ctx context.Context, base, apiKey, sender, to, fileURL, fileName, messageID string) (*sendResult, error) {
	return c.send(ctx, base, apiKey, outMessage{
		Sender:       sender,
		Destinations: []dest{{To: to}},
		Content:      fileContent{Type: contentTypeFile, MediaURL: fileURL, FileName: fileName},
		MessageID:    messageID,
	})
}

func (c *apiClient) SendTemplate(ctx context.Context, base, apiKey, sender, to, templateID, language string, params map[string]string, messageID string) (*sendResult, error) {
	return c.send(ctx, base, apiKey, outMessage{
		Sender:       sender,
		Destinations: []dest{{To: to}},
		Content:      templateContent{Type: contentTypeTemplate, TemplateID: templateID, Language: language, Parameters: params},
		MessageID:    messageID,
	})
}

// truncateBody bounds an InfoBip error body before it reaches logs/errors.
func truncateBody(body []byte) string {
	const limit = 512

	s := strings.TrimSpace(string(body))
	if len(s) > limit {
		return s[:limit]
	}

	return s
}

func (c *apiClient) send(ctx context.Context, base, apiKey string, msg outMessage) (*sendResult, error) {
	raw, err := json.Marshal(envelope{Messages: []outMessage{msg}})
	if err != nil {
		return nil, fmt.Errorf("viber_bm: marshal send: %w", err)
	}

	url := strings.TrimRight(base, "/") + sendMessagesPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", authScheme+" "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var out sendResponse
	if decErr := json.Unmarshal(body, &out); decErr != nil {
		return nil, fmt.Errorf("viber_bm: decode send response (http %d): %w", resp.StatusCode, decErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if out.Error != nil && out.Error.ServiceException.Text != "" {
			return nil, fmt.Errorf("viber_bm: http %d: %s", resp.StatusCode, out.Error.ServiceException.Text)
		}

		// Validation errors don't always fit the serviceException envelope.
		return nil, fmt.Errorf("viber_bm: http %d: %s", resp.StatusCode, truncateBody(body))
	}

	if len(out.Messages) == 0 {
		return nil, errors.New("viber_bm: empty send response")
	}

	m := out.Messages[0]
	if strings.EqualFold(m.Status.GroupName, "REJECTED") {
		c.logger.WarnContext(ctx, "viber_bm send rejected", "group", m.Status.GroupName, "description", m.Status.Description)

		return nil, fmt.Errorf("viber_bm: message rejected: %s", m.Status.Description)
	}

	return &sendResult{BulkID: out.BulkID, MessageID: m.MessageID, GroupName: m.Status.GroupName}, nil
}
