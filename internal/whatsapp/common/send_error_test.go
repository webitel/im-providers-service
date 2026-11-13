package common

import (
	"testing"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestSendErrorCode(t *testing.T) {
	cases := []struct {
		name    string
		code    int
		details string
		want    string
	}{
		{"media upload", 131053, "Downloading media from weblink failed with http code 404", sharedmodel.SendErrFileUnavailable},
		{"media upload size", 131053, "Media file size too big. Max file size we currently support: 100MB", sharedmodel.SendErrFileTooLarge},
		{"media download", 131052, "", sharedmodel.SendErrFileUnavailable},
		{"unsupported type", 131051, "", sharedmodel.SendErrUnsupportedMedia},
		{"undeliverable", 131026, "", sharedmodel.SendErrRecipientUnavailable},
		{"re-engagement", 131047, "", sharedmodel.SendErrRecipientUnavailable},
		{"recipient is sender", 131021, "", sharedmodel.SendErrRecipientUnavailable},
		{"rate limit", 130429, "", sharedmodel.SendErrChannelUnavailable},
		{"generic", 131000, "", sharedmodel.SendErrChannelUnavailable},
		{"overloaded", 131016, "", sharedmodel.SendErrChannelUnavailable},
		{"throughput", 80007, "", sharedmodel.SendErrChannelUnavailable},
		{"too many calls", 4, "", sharedmodel.SendErrChannelUnavailable},
		{"user request limit", 17, "", sharedmodel.SendErrChannelUnavailable},
		{"spam rate limit", 131048, "", sharedmodel.SendErrChannelUnavailable},
		{"access token expired", 190, "", sharedmodel.SendErrChannelUnavailable},
		{"pair rate limit", 131056, "", sharedmodel.SendErrChannelUnavailable},
		{"register rate limit", 133016, "", sharedmodel.SendErrChannelUnavailable},
		{"invalid parameter", 100, "", ""},
		{"unknown", 0, "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SendErrorCode(tc.code, tc.details); got != tc.want {
				t.Errorf("SendErrorCode(%d, %q) = %q, want %q", tc.code, tc.details, got, tc.want)
			}
		})
	}
}
