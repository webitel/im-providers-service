package viberbm

import (
	"errors"
	"fmt"
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

func TestSendErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"video too large", vibbmmodel.ErrVideoTooLarge, sharedmodel.SendErrFileTooLarge},
		{"file type rejected", vibbmmodel.ErrFileTypeUnsupported, sharedmodel.SendErrUnsupportedMedia},
		{"audio rejected", vibbmmodel.ErrAudioUnsupported, sharedmodel.SendErrUnsupportedMedia},
		{"video too long", vibbmmodel.ErrVideoTooLong, sharedmodel.SendErrUnsupportedMedia},
		{"preview unavailable", fmt.Errorf("%w: storage file id %q", vibbmmodel.ErrVideoPreviewUnavailable, "x"), sharedmodel.SendErrFileUnavailable},
		{"session window", vibbmmodel.ErrSessionWindowRequired, sharedmodel.SendErrRecipientUnavailable},
		{"http 503", &httpError{StatusCode: 503, Text: "unavailable"}, sharedmodel.SendErrChannelUnavailable},
		{"http 429", &httpError{StatusCode: 429, Text: "slow down"}, sharedmodel.SendErrChannelUnavailable},
		{"http 400", &httpError{StatusCode: 400, Text: "bad request"}, ""},
		{"media missing", vibbmmodel.ErrMediaMissing, ""},
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

func TestStatusErrorCode(t *testing.T) {
	cases := []struct {
		group, name, desc string
		want              string
	}{
		{"EXPIRED", "EXPIRED_EXPIRED", "Message expired", sharedmodel.SendErrRecipientUnavailable},
		{"REJECTED", "REJECTED_NOT_VIBER_USER", "", sharedmodel.SendErrRecipientUnavailable},
		{"UNDELIVERABLE", "UNDELIVERABLE_NOT_DELIVERED", "Recipient is not a Viber user", sharedmodel.SendErrRecipientUnavailable},
		{"REJECTED", "REJECTED_DESTINATION", "Destination blocked the sender", sharedmodel.SendErrRecipientUnavailable},
		{"REJECTED", "REJECTED_OPERATOR", "operator rejected", sharedmodel.SendErrSendFailed},
		{"UNDELIVERABLE", "UNDELIVERABLE_REJECTED_OPERATOR", "", sharedmodel.SendErrSendFailed},
		{"REJECTED", "REJECTED_SENDER", "Sender blocked", sharedmodel.SendErrChannelUnavailable},
		{"REJECTED", "REJECTED_SOURCE", "", sharedmodel.SendErrChannelUnavailable},
		{"REJECTED", "REJECTED_NOT_ENOUGH_CREDITS", "Not enough credits", sharedmodel.SendErrChannelUnavailable},
		{"REJECTED", "REJECTED_DESTINATION_NOT_REGISTERED", "", sharedmodel.SendErrRecipientUnavailable},
		{"REJECTED", "REJECTED_DND", "", sharedmodel.SendErrRecipientUnavailable},
		{"UNDELIVERABLE", "UNDELIVERABLE_NOT_DELIVERED", "User blocked the sender", sharedmodel.SendErrRecipientUnavailable},
		{"UNDELIVERABLE", "UNDELIVERABLE_NOT_DELIVERED", "Sender is blocked by Viber", sharedmodel.SendErrChannelUnavailable},
		{"REJECTED", "REJECTED_OPERATOR", "Service account is blocked", sharedmodel.SendErrChannelUnavailable},
		{"REJECTED", "REJECTED_OPERATOR", "Blocked", sharedmodel.SendErrSendFailed},
	}

	for _, tc := range cases {
		if got := statusErrorCode(tc.group, tc.name, tc.desc); got != tc.want {
			t.Errorf("statusErrorCode(%q, %q, %q) = %q, want %q", tc.group, tc.name, tc.desc, got, tc.want)
		}
	}
}

func TestStatusErrorMessage(t *testing.T) {
	cases := []struct{ name, desc, want string }{
		{"REJECTED_NOT_VIBER_USER", "not a viber user", "REJECTED_NOT_VIBER_USER: not a viber user"},
		{"EXPIRED", "", "EXPIRED"},
		{"", "timed out", "timed out"},
	}

	for _, tc := range cases {
		if got := statusErrorMessage(tc.name, tc.desc); got != tc.want {
			t.Errorf("statusErrorMessage(%q, %q) = %q, want %q", tc.name, tc.desc, got, tc.want)
		}
	}
}

func TestToResponse_TagsSendErrors(t *testing.T) {
	_, err := toResponse(nil, "380000000000", vibbmmodel.ErrVideoTooLarge)

	if !errors.Is(err, vibbmmodel.ErrVideoTooLarge) {
		t.Errorf("sentinel lost: %v", err)
	}

	if got := sharedmodel.ClassifySendError(err); got != sharedmodel.SendErrFileTooLarge {
		t.Errorf("classified as %q", got)
	}
}
