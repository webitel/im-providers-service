package webhook

import (
	"context"
	"encoding/json"

	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type WhatsAppBusinessAccountWebHook struct {
}

func (webHook *WhatsAppBusinessAccountWebHook) Type() string { return "whatsapp" }

func (webHook *WhatsAppBusinessAccountWebHook) HandleWebhook(ctx context.Context, paylaod []byte) error {
	return nil
}

func unmarshalWebhookPayload[T any](payload []byte) (T, error) {
	var result T

	if err := json.Unmarshal(payload, &result); err != nil {
		return result, errors.Internal("unmarshaling whatsapp web-hook payload", errors.WithCause(err), errors.WithID("webhook.model.unmarshal_webhook_payload"))
	}

	return result, nil
}
