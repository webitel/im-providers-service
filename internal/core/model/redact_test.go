package model

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		secret string
		want   string
	}{
		{
			name:   "telegram method url",
			in:     `Post "https://api.telegram.org/bot123456:test-token-telegram/sendMessage": dial tcp: i/o timeout`,
			secret: "test-token-telegram",
			want:   `Post "https://api.telegram.org/bot<redacted>/sendMessage": dial tcp: i/o timeout`,
		},
		{
			name:   "telegram file url with an unusual token shape",
			in:     `GET https://api.telegram.org/file/bottest-token-file/documents/a.pdf failed`,
			secret: "test-token-file",
			want:   `GET https://api.telegram.org/file/bot<redacted>/documents/a.pdf failed`,
		},
		{
			name:   "bot path on another host",
			in:     `Post "https://tg-proxy.example.com/bot42:test-token-proxy/sendPhoto": EOF`,
			secret: "test-token-proxy",
			want:   `Post "https://tg-proxy.example.com/bot<redacted>/sendPhoto": EOF`,
		},
		{
			name:   "url userinfo",
			in:     `Post "https://partner:test-token-basic@hooks.example.com/im": connection refused`,
			secret: "test-token-basic",
			want:   `Post "https://<redacted>@hooks.example.com/im": connection refused`,
		},
		{
			name:   "graph access_token query",
			in:     `Get "https://graph.facebook.com/v21.0/me?fields=id&access_token=test-token-graph&appsecret_proof=test-token-proof": EOF`,
			secret: "test-token-",
			want:   `Get "https://graph.facebook.com/v21.0/me?fields=id&access_token=<redacted>&appsecret_proof=<redacted>": EOF`,
		},
		{
			name:   "custom webhook token query",
			in:     `Post "https://hooks.example.com/im?token=test-token-hook": timeout`,
			secret: "test-token-hook",
			want:   `Post "https://hooks.example.com/im?token=<redacted>": timeout`,
		},
		{
			name:   "bearer echo",
			in:     `upstream said: Authorization: Bearer test-token-bearer is invalid`,
			secret: "test-token-bearer",
			want:   `upstream said: Authorization: Bearer <redacted> is invalid`,
		},
		{
			name:   "viber auth token echo",
			in:     `{"X-Viber-Auth-Token":"test-token-viber","status":2}`,
			secret: "test-token-viber",
			want:   `{"X-Viber-Auth-Token":"<redacted>","status":2}`,
		},
		{
			name: "plain provider text untouched",
			in:   "telegram: sendDocument failed (http 400, code 400): Bad Request: file is too big",
			want: "telegram: sendDocument failed (http 400, code 400): Bad Request: file is too big",
		},
		{
			name: "plain url untouched",
			in:   `Post "https://hooks.example.com/bots/list?page=2": EOF`,
			want: `Post "https://hooks.example.com/bots/list?page=2": EOF`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := RedactSecrets(tc.in)
			if got != tc.want {
				t.Fatalf("RedactSecrets() = %q, want %q", got, tc.want)
			}

			if tc.secret != "" && strings.Contains(got, tc.secret) {
				t.Fatalf("secret %q leaked: %q", tc.secret, got)
			}
		})
	}
}
