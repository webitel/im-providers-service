package facebook

import (
	"errors"

	"github.com/sony/gobreaker"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider/graphapi"
)

// sendErrorCode maps a Messenger send failure to a normalized code, "" when
// the error says nothing specific. Backoff exhaustion surfaces as the last
// attempt's error, so a 5xx APIError covers that case too.
func sendErrorCode(err error) string {
	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return sharedmodel.SendErrChannelUnavailable
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return graphapi.SendErrorCode(apiErr.StatusCode, []byte(apiErr.Body))
	}

	return ""
}

func tagSendError(err error) error {
	return sharedmodel.NewSendError(sendErrorCode(err), err)
}
