package instagram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ValidateSignature implements iface.SignatureValidator.
// It validates the X-Hub-Signature-256 header sent by Instagram on every webhook POST.
func (p *instagramProvider) ValidateSignature(ctx context.Context, headers http.Header, body []byte) error {
	header := headers.Get("X-Hub-Signature-256")
	if header == "" {
		return errors.New("missing X-Hub-Signature-256 header")
	}

	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return errors.New("invalid signature format")
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
		return errors.New("signature mismatch")
	}

	return nil
}
