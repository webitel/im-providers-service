package custom

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

const signHeader = "X-Webitel-Sign"

const validateSignatureErrID = "custom.signature.validate_signature"

// sign is the shared authentication of the channel, computed identically in
// both directions: hex-encoded HMAC-SHA256 of the raw body under the gate secret.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)

	return hex.EncodeToString(mac.Sum(nil))
}

func (p *customProvider) ValidateSignature(ctx context.Context, headers http.Header, body []byte) error {
	gate, err := p.resolveGate(ctx, p.webhookURI(ctx))
	if err != nil {
		return errors.Wrap(err, errors.WithID(validateSignatureErrID))
	}

	if gate == nil || !gate.Enabled {
		return errors.NotFound("custom: gate missing or disabled", errors.WithID(validateSignatureErrID))
	}

	if err := allowedSource(gate.AllowedIPs, headers); err != nil {
		return err
	}

	given := strings.TrimSpace(headers.Get(signHeader))
	if given == "" {
		return errors.Append(custommodel.ErrSignatureInvalid, signHeader+" header missing",
			errors.WithID(validateSignatureErrID))
	}

	expected := sign(body, gate.AppSecret)
	if !hmac.Equal([]byte(strings.ToLower(given)), []byte(expected)) {
		return custommodel.ErrSignatureInvalid
	}

	return nil
}

// allowedSource checks the peer address against the gate allowlist
func allowedSource(allowed []string, headers http.Header) error {
	if len(allowed) == 0 {
		return nil
	}

	remote := sourceIP(headers)
	if remote == nil {
		return errors.Append(custommodel.ErrSourceNotAllowed, "source address unknown",
			errors.WithID("custom.signature.allowed_source"))
	}

	for _, entry := range allowed {
		if strings.TrimSpace(entry) == "" {
			continue
		}

		network, err := custommodel.ParseAllowedIP(entry)
		if err != nil {
			continue
		}

		if network.Contains(remote) {
			return nil
		}
	}

	return errors.Append(custommodel.ErrSourceNotAllowed, remote.String(),
		errors.WithID("custom.signature.allowed_source"))
}

func sourceIP(headers http.Header) net.IP {
	if v := strings.TrimSpace(headers.Get("X-Real-IP")); v != "" {
		if ip := net.ParseIP(v); ip != nil {
			return ip
		}
	}

	// The left-most entry of X-Forwarded-For is the original client; the
	// entries after it are the proxies it passed through.
	if v := headers.Get("X-Forwarded-For"); v != "" {
		for _, part := range strings.Split(v, ",") {
			if ip := net.ParseIP(strings.TrimSpace(part)); ip != nil {
				return ip
			}
		}
	}

	return nil
}
