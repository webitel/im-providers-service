package model

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	kiterrors "github.com/webitel/webitel-go-kit/pkg/errors"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

var _ net.Error = timeoutErr{}

func TestClassifySendError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"tagged", NewSendError(SendErrFileTooLarge, errors.New("too big")), SendErrFileTooLarge},
		{"tagged deep in chain", fmt.Errorf("send: %w", NewSendError(SendErrRecipientUnavailable, errors.New("blocked"))), SendErrRecipientUnavailable},
		{"tag beats transient cause", NewSendError(SendErrFileUnavailable, context.DeadlineExceeded), SendErrFileUnavailable},
		{"context deadline", fmt.Errorf("post: %w", context.DeadlineExceeded), SendErrChannelUnavailable},
		{"net timeout", &net.OpError{Op: "dial", Err: timeoutErr{}}, SendErrChannelUnavailable},
		{"grpc unavailable", status.Error(codes.Unavailable, "down"), SendErrChannelUnavailable},
		{"grpc deadline", status.Error(codes.DeadlineExceeded, "slow"), SendErrChannelUnavailable},
		{"grpc resource exhausted", status.Error(codes.ResourceExhausted, "rate"), SendErrChannelUnavailable},
		{"wrapped grpc unavailable", fmt.Errorf("x: %w", status.Error(codes.Unavailable, "down")), SendErrChannelUnavailable},
		{"grpc invalid argument", status.Error(codes.InvalidArgument, "bad"), SendErrSendFailed},
		{"kit unavailable", kiterrors.Unavailable("unreachable"), SendErrChannelUnavailable},
		{"kit appended unavailable", kiterrors.Append(kiterrors.Unavailable("unreachable"), "status 502"), SendErrChannelUnavailable},
		{"kit resource exhausted", kiterrors.New("rate", kiterrors.WithCode(codes.ResourceExhausted)), SendErrChannelUnavailable},
		{"kit aborted", kiterrors.Aborted("rejected"), SendErrSendFailed},
		{"context canceled", context.Canceled, SendErrSendFailed},
		{"plain", errors.New("boom"), SendErrSendFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifySendError(tc.err); got != tc.want {
				t.Errorf("ClassifySendError(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestNewSendError(t *testing.T) {
	base := errors.New("provider said no")

	if NewSendError(SendErrSendFailed, nil) != nil {
		t.Error("nil error must stay nil")
	}

	if got := NewSendError("", base); got != base {
		t.Errorf("empty code must return err untouched, got %v", got)
	}

	wrapped := NewSendError(SendErrFileTooLarge, base)
	if wrapped.Error() != base.Error() {
		t.Errorf("message changed: %q", wrapped.Error())
	}

	if !errors.Is(wrapped, base) {
		t.Error("wrapped error must unwrap to the provider error")
	}

	if got := ClassifySendError(NewSendError(SendErrChannelUnavailable, wrapped)); got != SendErrFileTooLarge {
		t.Errorf("inner code must win, got %q", got)
	}
}

func TestSendErrCodeForHTTPStatus(t *testing.T) {
	cases := []struct {
		status int
		want   string
	}{
		{200, ""},
		{400, ""},
		{403, ""},
		{404, ""},
		{413, SendErrFileTooLarge},
		{429, SendErrChannelUnavailable},
		{500, SendErrChannelUnavailable},
		{503, SendErrChannelUnavailable},
	}

	for _, tc := range cases {
		if got := SendErrCodeForHTTPStatus(tc.status); got != tc.want {
			t.Errorf("SendErrCodeForHTTPStatus(%d) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestCheckFileSize(t *testing.T) {
	const limit = 5 << 20

	cases := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"unknown size", 0, false},
		{"negative size", -1, false},
		{"under limit", limit - 1, false},
		{"at limit", limit, false},
		{"over limit", limit + 1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckFileSize("photo", tc.size, limit)
			if (err != nil) != tc.wantErr {
				t.Fatalf("CheckFileSize(%d) = %v, wantErr %v", tc.size, err, tc.wantErr)
			}

			if tc.wantErr && ClassifySendError(err) != SendErrFileTooLarge {
				t.Errorf("classified as %q", ClassifySendError(err))
			}
		})
	}

	if err := CheckFileSize("file", 1<<40, 0); err != nil {
		t.Errorf("zero limit must disable the check, got %v", err)
	}
}

func TestIsSendErrCode(t *testing.T) {
	for _, code := range []string{
		SendErrFileTooLarge, SendErrFileUnavailable, SendErrUnsupportedMedia,
		SendErrRecipientUnavailable, SendErrChannelUnavailable, SendErrSendFailed,
	} {
		if !IsSendErrCode(code) {
			t.Errorf("IsSendErrCode(%q) = false", code)
		}
	}

	for _, code := range []string{"", "FILE_TOO_LARGE", "unknown chat", "callback_undelivered"} {
		if IsSendErrCode(code) {
			t.Errorf("IsSendErrCode(%q) = true", code)
		}
	}
}
