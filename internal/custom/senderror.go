package custom

import (
	"net/http"

	"github.com/webitel/webitel-go-kit/pkg/errors"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

// sendErrorCode maps a callback delivery failure to a normalized code, ""
// when the error says nothing specific. A code attached by the client from
// the HTTP answer takes precedence over this one.
func sendErrorCode(err error) string {
	switch {
	case errors.Is(err, custommodel.ErrCallbackRejected):
		return sharedmodel.SendErrSendFailed
	case errors.Is(err, custommodel.ErrCallbackFailed):
		return sharedmodel.SendErrChannelUnavailable
	default:
		return ""
	}
}

// callbackStatusCode maps a non-2xx callback answer. The external system may
// name a normalized code in the contract's "error" field; otherwise 4xx is a
// permanent refusal except for the statuses that say "later" or "too big".
func callbackStatusCode(statusCode int, reason string) string {
	if code := reasonCode(reason); code != "" {
		return code
	}

	switch {
	case statusCode == http.StatusRequestEntityTooLarge:
		return sharedmodel.SendErrFileTooLarge
	case statusCode == http.StatusRequestTimeout, statusCode == http.StatusTooManyRequests,
		statusCode >= http.StatusInternalServerError:
		return sharedmodel.SendErrChannelUnavailable
	case statusCode >= http.StatusBadRequest:
		return sharedmodel.SendErrSendFailed
	default:
		return ""
	}
}

// reasonCode returns reason when the external system answered with one of
// the normalized codes, "" otherwise.
func reasonCode(reason string) string {
	if sharedmodel.IsSendErrCode(reason) {
		return reason
	}

	return ""
}

func tagSendError(err error) error {
	return sharedmodel.NewSendError(sendErrorCode(err), err)
}

// undeliveredCode is the code of a message the retry queue gave up on: unless
// the last attempt said something specific, the endpoint was unreachable.
func undeliveredCode(cause error) string {
	var tagged *sharedmodel.SendError
	if errors.As(cause, &tagged) {
		return tagged.Code
	}

	if errors.Is(cause, custommodel.ErrCallbackRejected) {
		return sharedmodel.SendErrSendFailed
	}

	return sharedmodel.SendErrChannelUnavailable
}
