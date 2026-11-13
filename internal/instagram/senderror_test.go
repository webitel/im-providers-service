package instagram

import (
	"context"
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
		{"media type", fmt.Errorf("%w: got %q", ErrMediaTypeNotSupported, "file"), sharedmodel.SendErrUnsupportedMedia},
		{"circuit open", gobreaker.ErrOpenState, sharedmodel.SendErrChannelUnavailable},
		{"outside window", fmt.Errorf("ig send: %w", &APIError{StatusCode: 400, Body: `{"error":{"code":10,"error_subcode":2018278,"message":"outside of allowed window"}}`}), sharedmodel.SendErrRecipientUnavailable},
		{"server error", fmt.Errorf("ig send: %w", &APIError{StatusCode: 503, Body: ""}), sharedmodel.SendErrChannelUnavailable},
		{"bad param", &APIError{StatusCode: 400, Body: `{"error":{"code":100,"message":"(#100) Invalid parameter"}}`}, ""},
		{"text too long", ErrTextTooLong, ""},
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

func TestSendDocument_UnsupportedMedia(t *testing.T) {
	_, err := (&instagramProvider{}).SendDocument(context.Background(), &sharedmodel.Message{})

	if got := sharedmodel.ClassifySendError(err); got != sharedmodel.SendErrUnsupportedMedia {
		t.Errorf("classified as %q", got)
	}
}
