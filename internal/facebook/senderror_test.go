package facebook

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sony/gobreaker"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestSendErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"circuit open", gobreaker.ErrOpenState, sharedmodel.SendErrChannelUnavailable},
		{"half-open overflow", gobreaker.ErrTooManyRequests, sharedmodel.SendErrChannelUnavailable},
		{"outside window", fmt.Errorf("fb send: %w", &APIError{StatusCode: 400, Body: `{"error":{"code":10,"error_subcode":2018278,"message":"(#10) This message is sent outside of allowed window."}}`}), sharedmodel.SendErrRecipientUnavailable},
		{"user unavailable", &APIError{StatusCode: 400, Body: `{"error":{"code":551,"message":"(#551) This person isn't available right now."}}`}, sharedmodel.SendErrRecipientUnavailable},
		{"attachment fetch", &APIError{StatusCode: 400, Body: `{"error":{"code":100,"error_subcode":2018008,"message":"(#100) Failed to fetch the file from the url"}}`}, sharedmodel.SendErrFileUnavailable},
		{"server error", &APIError{StatusCode: 500, Body: `{"error":{"code":1,"message":"An unknown error has occurred."}}`}, sharedmodel.SendErrChannelUnavailable},
		{"bad param", &APIError{StatusCode: 400, Body: `{"error":{"code":100,"message":"(#100) Invalid parameter"}}`}, ""},
		{"token invalid", ErrTokenInvalid, ""},
		{"plain", errors.New("boom"), ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sendErrorCode(tc.err); got != tc.want {
				t.Errorf("sendErrorCode(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestWithRecipient_TagsError(t *testing.T) {
	apiErr := &APIError{StatusCode: 400, Body: `{"error":{"code":551,"message":"(#551) This person isn't available right now."}}`}

	_, err := withRecipient(nil, fmt.Errorf("fb send: %w", apiErr))("psid-1")

	var target *APIError
	if !errors.As(err, &target) {
		t.Fatal("tagged error must still expose the APIError")
	}

	if got := sharedmodel.ClassifySendError(err); got != sharedmodel.SendErrRecipientUnavailable {
		t.Errorf("classified as %q", got)
	}
}
