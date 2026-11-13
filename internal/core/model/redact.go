package model

import "regexp"

// Redacted replaces credentials scrubbed from error text.
const Redacted = "<redacted>"

// secretPatterns cover the credential shapes channel errors can carry: a
// *url.Error echoes the request URL, and provider bodies sometimes echo
// request fields back.
var secretPatterns = []struct {
	re   *regexp.Regexp
	repl string
}{
	// URL userinfo (custom webhook URLs may embed basic-auth credentials).
	{regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.\-]*://)[^/\s@"'<>]+@`), "${1}" + Redacted + "@"},
	// Telegram Bot API puts the bot token in the path: /bot<token>/method and
	// /file/bot<token>/path. https://core.telegram.org/bots/api#making-requests
	{regexp.MustCompile(`(api\.telegram\.org/(?:file/)?bot)[^/\s"'<>?#]+`), "${1}" + Redacted},
	{regexp.MustCompile(`(/bot)[0-9]+:[A-Za-z0-9_\-]+`), "${1}" + Redacted},
	// Token-like query parameters (Graph API access_token, appsecret_proof...).
	{regexp.MustCompile(`(?i)([?&;](?:access_token|token|auth_token|api_key|apikey|key|secret|client_secret|password|passwd|signature|sig|appsecret_proof|input_token)=)[^&\s"'#<>]+`), "${1}" + Redacted},
	// Header or JSON field echoes: Authorization, X-Viber-Auth-Token, auth_token.
	{regexp.MustCompile(`(?i)\b(bearer)\s+[A-Za-z0-9._~+/=\-]+`), "${1} " + Redacted},
	{regexp.MustCompile(`(?i)("?(?:x-viber-auth-token|auth_token|access_token|app_secret|appsecret)"?\s*[:=]\s*"?)[^"\s,&}]+`), "${1}" + Redacted},
}

// RedactSecrets scrubs credentials from free-form error text before it
// leaves the service (im-thread failure receipts, gRPC error messages).
func RedactSecrets(s string) string {
	for _, p := range secretPatterns {
		s = p.re.ReplaceAllString(s, p.repl)
	}

	return s
}
