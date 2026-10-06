package viberbm

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
)

// ValidateSignature verifies the static Authorization credential we provision
// on the subscription (InfoBip does not HMAC-sign forwards). An empty stored
// webhook_secret opts the gate out. Comparison is constant-time.
func (p *viberBMProvider) ValidateSignature(ctx context.Context, headers http.Header, _ []byte) error {
	uri := p.webhookURI(ctx)

	gate, err := p.repo.SelectByURI(ctx, uri)
	if err != nil {
		return err
	}

	if gate.WebhookSecret == "" {
		return nil
	}

	got := headers.Get("Authorization")
	if subtle.ConstantTimeCompare([]byte(got), []byte(gate.WebhookSecret)) != 1 {
		return errors.New("viber_bm signature: credential mismatch")
	}

	return nil
}
