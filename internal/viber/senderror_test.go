package viber

import (
	"errors"
	"fmt"
	"testing"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

func TestSendErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"receiver not subscribed", vibmodel.ErrReceiverNotSubscribed, sharedmodel.SendErrRecipientUnavailable},
		{"status mapped to not subscribed", statusToError(statusReceiverNotSubscribed, "not subscribed"), sharedmodel.SendErrRecipientUnavailable},
		{"no suitable device", statusToError(statusNoSuitableDevice, "receiverNoSuitableDevice"), sharedmodel.SendErrRecipientUnavailable},
		{"too many requests", statusToError(statusTooManyRequests, "tooManyRequests"), sharedmodel.SendErrChannelUnavailable},
		{"other api status", statusToError(3, "badData"), ""},
		{"http 502", &httpError{Path: "/send_message", StatusCode: 502}, sharedmodel.SendErrChannelUnavailable},
		{"http 413", fmt.Errorf("x: %w", &httpError{Path: "/send_message", StatusCode: 413}), sharedmodel.SendErrFileTooLarge},
		{"http 400", &httpError{Path: "/send_message", StatusCode: 400}, ""},
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

func TestTagSendError_KeepsSentinelAndMessage(t *testing.T) {
	err := tagSendError(vibmodel.ErrReceiverNotSubscribed)

	if !errors.Is(err, vibmodel.ErrReceiverNotSubscribed) {
		t.Error("tagged error must still match the sentinel")
	}

	if err.Error() != vibmodel.ErrReceiverNotSubscribed.Error() {
		t.Errorf("message changed: %q", err.Error())
	}

	if got := sharedmodel.ClassifySendError(err); got != sharedmodel.SendErrRecipientUnavailable {
		t.Errorf("classified as %q", got)
	}

	var apiErr *apiError
	if !errors.As(tagSendError(statusToError(3, "badData")), &apiErr) {
		t.Error("untagged api error must stay reachable for the media fallback")
	}
}

func TestFailedCallbackCode(t *testing.T) {
	cases := []struct {
		desc string
		want string
	}{
		{"", sharedmodel.SendErrSendFailed},
		{"some unknown reason", sharedmodel.SendErrSendFailed},
		{"Receiver is not subscribed", sharedmodel.SendErrRecipientUnavailable},
		{"user blocked the bot", sharedmodel.SendErrRecipientUnavailable},
		{"Failed to download media", sharedmodel.SendErrFileUnavailable},
		{"File size too big", sharedmodel.SendErrFileTooLarge},
		{"File is too large", sharedmodel.SendErrFileTooLarge},
		{"file size exceeds the limit", sharedmodel.SendErrFileTooLarge},
		{"Maximum size of 50MB reached", sharedmodel.SendErrFileTooLarge},
		{"Failed to download media: file size limit reached", sharedmodel.SendErrFileTooLarge},
		{"Failed to download the file, invalid size header", sharedmodel.SendErrFileUnavailable},
		{"Could not fetch media url", sharedmodel.SendErrFileUnavailable},
		{"Invalid size parameter", sharedmodel.SendErrSendFailed},
		{"Thumbnail size is invalid", sharedmodel.SendErrSendFailed},
		{"Rate limit exceeded", sharedmodel.SendErrSendFailed},
	}

	for _, tc := range cases {
		if got := failedCallbackCode(tc.desc); got != tc.want {
			t.Errorf("failedCallbackCode(%q) = %q, want %q", tc.desc, got, tc.want)
		}
	}
}

func TestCallbackTime(t *testing.T) {
	if got := callbackTime(1_770_000_000_000); !got.Equal(time.UnixMilli(1_770_000_000_000)) {
		t.Errorf("callbackTime = %v", got)
	}

	if got := callbackTime(0); time.Since(got) > time.Minute {
		t.Errorf("zero timestamp must fall back to now, got %v", got)
	}
}
