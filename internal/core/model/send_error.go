package model

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	kiterrors "github.com/webitel/webitel-go-kit/pkg/errors"
)

// Normalized outbound send failure codes reported to im-thread as
// FailureReceipt.error_code. The agent UI keys its user-facing text on them,
// so the values are a contract: never rename, only add.
const (
	SendErrFileTooLarge         = "file_too_large"
	SendErrFileUnavailable      = "file_unavailable"
	SendErrUnsupportedMedia     = "unsupported_media"
	SendErrRecipientUnavailable = "recipient_unavailable"
	SendErrChannelUnavailable   = "channel_unavailable"
	SendErrSendFailed           = "send_failed"
)

// IsSendErrCode reports whether code is one of the normalized SendErr* values.
func IsSendErrCode(code string) bool {
	switch code {
	case SendErrFileTooLarge, SendErrFileUnavailable, SendErrUnsupportedMedia,
		SendErrRecipientUnavailable, SendErrChannelUnavailable, SendErrSendFailed:
		return true
	default:
		return false
	}
}

// SendError tags a channel send failure with a normalized code while keeping
// the provider error (and its message) intact for errors.Is/As and logs.
type SendError struct {
	Code string
	Err  error
}

func (e *SendError) Error() string {
	if e.Err == nil {
		return e.Code
	}

	return e.Err.Error()
}

func (e *SendError) Unwrap() error { return e.Err }

// NewSendError wraps err with code. A nil err stays nil, an empty code leaves
// err untouched, and an err that already carries a code keeps it: the layer
// closest to the provider knows the failure best.
func NewSendError(code string, err error) error {
	if err == nil || code == "" {
		return err
	}

	var existing *SendError
	if errors.As(err, &existing) {
		return err
	}

	return &SendError{Code: code, Err: err}
}

// CheckFileSize rejects a file whose known size is over the channel limit
// before any provider call, so the agent gets file_too_large instead of a
// vague fetch failure from the platform. size <= 0 means unknown and passes.
func CheckFileSize(what string, size, limit int64) error {
	if size <= 0 || limit <= 0 || size <= limit {
		return nil
	}

	return &SendError{
		Code: SendErrFileTooLarge,
		Err:  fmt.Errorf("%s of %d bytes exceeds the channel limit of %d bytes", what, size, limit),
	}
}

// SendErrCodeForHTTPStatus maps the HTTP status of a failed channel API call
// to a code when the status alone is conclusive, "" otherwise.
func SendErrCodeForHTTPStatus(statusCode int) string {
	switch {
	case statusCode == http.StatusRequestEntityTooLarge:
		return SendErrFileTooLarge
	case statusCode == http.StatusTooManyRequests, statusCode >= http.StatusInternalServerError:
		return SendErrChannelUnavailable
	default:
		return ""
	}
}

// ClassifySendError maps any send error to a normalized code. Errors tagged
// via SendError win; otherwise transport-level transient failures become
// channel_unavailable and everything else send_failed.
func ClassifySendError(err error) string {
	if err == nil {
		return ""
	}

	var se *SendError
	if errors.As(err, &se) && se.Code != "" {
		return se.Code
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return SendErrChannelUnavailable
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return SendErrChannelUnavailable
	}

	if code := GRPCCode(err); code == codes.Unavailable || code == codes.DeadlineExceeded || code == codes.ResourceExhausted {
		return SendErrChannelUnavailable
	}

	return SendErrSendFailed
}

// GRPCCode reads the gRPC code from a status error or, failing that, from a
// webitel-go-kit error, which carries its code as a value instead of a status.
func GRPCCode(err error) codes.Code {
	if s, ok := status.FromError(err); ok {
		return s.Code()
	}

	if code, ok := kiterrors.Value(err, kiterrors.ErrKeyCode).(codes.Code); ok {
		return code
	}

	return codes.Unknown
}
