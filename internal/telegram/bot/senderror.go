package bot

import (
	"net/http"
	"strings"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	tgclient "github.com/webitel/im-providers-service/internal/telegram/bot/client"
)

// Bot API answers carry no stable machine codes beyond the HTTP status, so the
// description text is matched; fragments are lowercase.
// https://core.telegram.org/bots/api#making-requests
var (
	tooLargeMarkers = []string{"too big", "request entity too large"}

	fileUnavailableMarkers = []string{
		"failed to get http url content",
		"wrong file identifier",
		"wrong type of the web page content",
		"webpage_",
	}

	recipientMarkers = []string{
		"bot was blocked",
		"user is deactivated",
		"bot can't initiate",
		"chat not found",
		"bot was kicked",
	}
)

// sendErrorCode maps a Telegram Bot API send failure to a normalized code,
// "" when the error is not a Bot API answer.
func sendErrorCode(err error) string {
	var apiErr *tgclient.APIError
	if !errors.As(err, &apiErr) {
		return ""
	}

	desc := strings.ToLower(apiErr.Description)

	switch {
	case apiErr.StatusCode == http.StatusRequestEntityTooLarge || containsAny(desc, tooLargeMarkers):
		return coremodel.SendErrFileTooLarge
	case containsAny(desc, fileUnavailableMarkers):
		return coremodel.SendErrFileUnavailable
	case apiErr.StatusCode == http.StatusForbidden || containsAny(desc, recipientMarkers):
		return coremodel.SendErrRecipientUnavailable
	// 401 is a revoked or wrong bot token: the channel is down, not the message.
	case apiErr.StatusCode == http.StatusUnauthorized, apiErr.StatusCode == http.StatusTooManyRequests,
		apiErr.StatusCode >= http.StatusInternalServerError:
		return coremodel.SendErrChannelUnavailable
	default:
		return ""
	}
}

func tagSendError(err error) error {
	return coremodel.NewSendError(sendErrorCode(err), err)
}

func containsAny(s string, fragments []string) bool {
	for _, f := range fragments {
		if strings.Contains(s, f) {
			return true
		}
	}

	return false
}
