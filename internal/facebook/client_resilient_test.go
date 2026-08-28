package facebook

import (
	"context"
	"errors"
	"fmt"
	"testing"

	backoff "github.com/cenkalti/backoff/v3"
)

func TestRetryable(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantPermanent bool
	}{
		{name: "nil", err: nil, wantPermanent: false},
		{name: "token invalid is permanent", err: ErrTokenInvalid, wantPermanent: true},
		{name: "context canceled is permanent", err: context.Canceled, wantPermanent: true},
		{name: "context deadline is permanent", err: context.DeadlineExceeded, wantPermanent: true},
		{
			name:          "400 param error is permanent",
			err:           fmt.Errorf("fb send: %w", &APIError{StatusCode: 400, Body: "(#100) You cannot send messages to this id"}),
			wantPermanent: true,
		},
		{
			name:          "404 is permanent",
			err:           &APIError{StatusCode: 404, Body: "not found"},
			wantPermanent: true,
		},
		{
			name:          "429 rate limit is transient",
			err:           &APIError{StatusCode: 429, Body: "rate limited"},
			wantPermanent: false,
		},
		{
			name:          "500 is transient",
			err:           &APIError{StatusCode: 500, Body: "server error"},
			wantPermanent: false,
		},
		{
			name:          "plain error is transient",
			err:           errors.New("connection reset"),
			wantPermanent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := retryable(tt.err)
			if tt.err == nil {
				if got != nil {
					t.Fatalf("retryable(nil) = %v, want nil", got)
				}
				return
			}

			var perm *backoff.PermanentError
			isPermanent := errors.As(got, &perm)
			if isPermanent != tt.wantPermanent {
				t.Fatalf("permanent = %v, want %v (err=%v)", isPermanent, tt.wantPermanent, got)
			}

			// The original error must always be preserved for the caller/logs.
			if !errors.Is(got, tt.err) {
				t.Errorf("retryable dropped the original error: got %v, want it to wrap %v", got, tt.err)
			}
		})
	}
}
