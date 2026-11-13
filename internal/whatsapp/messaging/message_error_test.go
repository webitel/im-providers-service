package messaging

import (
	"testing"

	"google.golang.org/grpc/codes"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func TestMessageSendError_ToSendError(t *testing.T) {
	cases := []struct {
		name     string
		code     int
		details  string
		wantCode string
		wantGRPC codes.Code
	}{
		{"unsupported type", 131051, "", sharedmodel.SendErrUnsupportedMedia, codes.Internal},
		{"media size", 131053, "Media file size too big", sharedmodel.SendErrFileTooLarge, codes.Internal},
		{"re-engagement", 131047, "", sharedmodel.SendErrRecipientUnavailable, codes.Internal},
		{"throttled", 4, "", sharedmodel.SendErrChannelUnavailable, codes.ResourceExhausted},
		{"bad parameter", 100, "", sharedmodel.SendErrSendFailed, codes.InvalidArgument},
		{"unmapped code falls back to send_failed", 368, "", sharedmodel.SendErrSendFailed, codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := MessageSendError{Code: tc.code, Message: "meta said no"}
			in.ErrorData.Details = tc.details

			err := in.toSendError("message.manager.send")

			if got := sharedmodel.ClassifySendError(err); got != tc.wantCode {
				t.Errorf("code = %q, want %q", got, tc.wantCode)
			}

			if got := sharedmodel.GRPCCode(err); got != tc.wantGRPC {
				t.Errorf("grpc code = %s, want %s", got, tc.wantGRPC)
			}
		})
	}
}
