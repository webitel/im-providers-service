package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/provider/facebook/graph"
	"github.com/webitel/im-providers-service/internal/provider/facebook/payload"
)

type UserProfile struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name" graph:"first_name"`
	LastName   string `json:"last_name" graph:"last_name"`
	ProfilePic string `json:"profile_pic" graph:"profile_pic"`
	Locale     string `json:"locale"`
	Timezone   int    `json:"timezone"`
}

type Client struct {
	logger *slog.Logger
	apiURL string
}

func NewClient(l *slog.Logger) *Client {
	return &Client{logger: l.With("pkg", "fb.client"), apiURL: GraphBaseURL}
}

func (c *Client) GetUserProfile(ctx context.Context, psid, token string) (*UserProfile, error) {
	u, err := graph.NewQuery(c.apiURL, psid).
		WithFields(graph.ID, graph.FirstName, graph.LastName, graph.ProfilePic, graph.Locale, graph.Timezone).
		WithToken(token).
		Build()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fb api error: %d, body: %s", resp.StatusCode, b)
	}

	var p UserProfile
	return &p, json.NewDecoder(resp.Body).Decode(&p)
}

func (c *Client) send(ctx context.Context, token string, body graph.OutboundPayload) (*model.MessageResponse, error) {
	u := fmt.Sprintf("%s/me/messages?access_token=%s", c.apiURL, token)
	raw, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewBuffer(raw))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fb send error: %d, body: %s", resp.StatusCode, b)
	}

	var res struct {
		ID string `json:"message_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return &model.MessageResponse{ID: res.ID}, nil
}

func (c *Client) ParseWebhook(data []byte) (*payload.WebhookRequest, error) {
	var r payload.WebhookRequest
	return &r, json.Unmarshal(data, &r)
}

func (c *Client) SendText(ctx context.Context, token, psid, text string) (*model.MessageResponse, error) {
	return c.send(ctx, token, graph.NewTextRequest(psid, text))
}

func (c *Client) SendMedia(ctx context.Context, token, psid, mType, url string) (*model.MessageResponse, error) {
	return c.send(ctx, token, graph.NewMediaRequest(psid, mType, url))
}
