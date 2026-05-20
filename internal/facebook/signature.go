package facebook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ValidateSignature implements iface.SignatureValidator.
// It validates the X-Hub-Signature-256 header sent by Facebook on every webhook POST.
func (p *facebookProvider) ValidateSignature(ctx context.Context, header string, body []byte) error {
	if header == "" {
		return fmt.Errorf("missing X-Hub-Signature-256 header")
	}

	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return fmt.Errorf("invalid signature format")
	}

	uri := p.webhookURI(ctx)
	app, err := p.metaAppRepo.SelectByURI(ctx, uri)
	if err != nil {
		return fmt.Errorf("signature: app lookup failed: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(app.AppSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	given := strings.TrimPrefix(header, prefix)
	if !hmac.Equal([]byte(given), []byte(expected)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}
