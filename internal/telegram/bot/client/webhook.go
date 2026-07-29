package client

import (
	"context"
	"encoding/json"
)

const (
	setWebhookMethod    = "setWebhook"
	deleteWebhookMethod = "deleteWebhook"
)

// https://core.telegram.org/bots/api#setwebhook
type SetWebhookRequest struct {
	URL         string `json:"url"`
	SecretToken string `json:"secret_token,omitempty"`
}

func (c *Client) SetWebhook(ctx context.Context, token, url, secretToken string) error {
	body, err := json.Marshal(SetWebhookRequest{URL: url, SecretToken: secretToken})
	if err != nil {
		return err
	}

	return c.callMethod(ctx, setWebhookMethod, token, body, nil)
}

// https://core.telegram.org/bots/api#deletewebhook
func (c *Client) DeleteWebhook(ctx context.Context, token string) error {
	return c.callMethod(ctx, deleteWebhookMethod, token, []byte("{}"), nil)
}
