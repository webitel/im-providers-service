package bot

import (
	"errors"
	"fmt"
	"testing"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	tgclient "github.com/webitel/im-providers-service/internal/telegram/bot/client"
)

func apiErr(status int, desc string) error {
	return &tgclient.APIError{Method: "sendDocument", StatusCode: status, ErrorCode: status, Description: desc}
}

func TestSendErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"413 status", apiErr(413, "Request Entity Too Large"), coremodel.SendErrFileTooLarge},
		{"file too big", apiErr(400, "Bad Request: file is too big"), coremodel.SendErrFileTooLarge},
		{"url fetch failed", apiErr(400, "Bad Request: failed to get HTTP URL content"), coremodel.SendErrFileUnavailable},
		{"wrong file id", apiErr(400, "Bad Request: wrong file identifier/HTTP URL specified"), coremodel.SendErrFileUnavailable},
		{"wrong content type", apiErr(400, "Bad Request: wrong type of the web page content"), coremodel.SendErrFileUnavailable},
		{"webpage error", apiErr(400, "Bad Request: WEBPAGE_CURL_FAILED"), coremodel.SendErrFileUnavailable},
		{"bot blocked", apiErr(403, "Forbidden: bot was blocked by the user"), coremodel.SendErrRecipientUnavailable},
		{"user deactivated", apiErr(403, "Forbidden: user is deactivated"), coremodel.SendErrRecipientUnavailable},
		{"cannot initiate", apiErr(403, "Forbidden: bot can't initiate conversation with a user"), coremodel.SendErrRecipientUnavailable},
		{"chat not found", apiErr(400, "Bad Request: chat not found"), coremodel.SendErrRecipientUnavailable},
		{"rate limited", apiErr(429, "Too Many Requests: retry after 5"), coremodel.SendErrChannelUnavailable},
		{"server error", apiErr(502, "Bad Gateway"), coremodel.SendErrChannelUnavailable},
		{"revoked bot token", apiErr(401, "Unauthorized"), coremodel.SendErrChannelUnavailable},
		{"wrapped api error", fmt.Errorf("send: %w", apiErr(403, "Forbidden: bot was blocked by the user")), coremodel.SendErrRecipientUnavailable},
		{"other bad request", apiErr(400, "Bad Request: can't parse entities"), ""},
		{"not an api error", errors.New("dial tcp: connection refused"), ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sendErrorCode(tc.err); got != tc.want {
				t.Errorf("sendErrorCode(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestTagSendError_KeepsAPIError(t *testing.T) {
	err := tagSendError(apiErr(403, "Forbidden: bot was blocked by the user"))

	var target *tgclient.APIError
	if !errors.As(err, &target) {
		t.Fatal("tagged error must still expose the APIError")
	}

	if got := coremodel.ClassifySendError(err); got != coremodel.SendErrRecipientUnavailable {
		t.Errorf("classified as %q", got)
	}
}
