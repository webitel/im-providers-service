package viber

import (
	"errors"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

// sendErrorCode maps a Viber send failure to a normalized code, "" when the
// error says nothing specific. Status codes:
// https://developers.viber.com/docs/api/rest-bot-api/#error-codes
func sendErrorCode(err error) string {
	if errors.Is(err, vibmodel.ErrReceiverNotSubscribed) {
		return sharedmodel.SendErrRecipientUnavailable
	}

	var apiErr *apiError
	if errors.As(err, &apiErr) {
		switch apiErr.Status {
		case statusNoSuitableDevice:
			return sharedmodel.SendErrRecipientUnavailable
		case statusTooManyRequests:
			return sharedmodel.SendErrChannelUnavailable
		}

		return ""
	}

	var httpErr *httpError
	if errors.As(err, &httpErr) {
		return sharedmodel.SendErrCodeForHTTPStatus(httpErr.StatusCode)
	}

	return ""
}

func tagSendError(err error) error {
	return sharedmodel.NewSendError(sendErrorCode(err), err)
}

// failedCallbackTooLarge are size phrases specific enough not to catch an
// unrelated desc that merely mentions a size ("invalid size parameter").
var failedCallbackTooLarge = []string{
	"too large", "too big", "max size", "maximum size", "file size limit", "size limit",
}

// failedCallbackCode maps the free-text desc of a "failed" delivery callback.
// Viber documents no stable values for it, so only unambiguous wording is
// mapped and the raw desc travels as the error message.
func failedCallbackCode(desc string) string {
	d := strings.ToLower(desc)

	switch {
	case containsAny(d, failedCallbackTooLarge),
		strings.Contains(d, "exceed") && (strings.Contains(d, "size") || strings.Contains(d, "file") || strings.Contains(d, "bytes")):
		return sharedmodel.SendErrFileTooLarge
	case strings.Contains(d, "download"), strings.Contains(d, "media url"), strings.Contains(d, "fetch"):
		return sharedmodel.SendErrFileUnavailable
	case strings.Contains(d, "not subscribed"), strings.Contains(d, "unsubscribed"), strings.Contains(d, "blocked"),
		strings.Contains(d, "not registered"), strings.Contains(d, "no suitable device"):
		return sharedmodel.SendErrRecipientUnavailable
	default:
		return sharedmodel.SendErrSendFailed
	}
}

func containsAny(s string, fragments []string) bool {
	for _, f := range fragments {
		if strings.Contains(s, f) {
			return true
		}
	}

	return false
}
