package client

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
)

const testToken = "123456:test-token-telegram"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestCallMethod_TransportErrorRedactsToken(t *testing.T) {
	c := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, &net.OpError{Op: "dial", Net: "tcp", Err: timeoutErr{}}
	})}}

	err := c.callMethod(context.Background(), sendTextMethod, testToken, []byte("{}"), nil)
	if err == nil {
		t.Fatal("expected a transport error")
	}

	if strings.Contains(err.Error(), "test-token-telegram") {
		t.Fatalf("token leaked: %q", err.Error())
	}

	if !strings.Contains(err.Error(), "/bot"+redactedToken+"/"+sendTextMethod) {
		t.Errorf("redacted URL missing from %q", err.Error())
	}

	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Errorf("timeout classification lost: %v", err)
	}
}

func TestCallMethod_ContextErrorKeepsCause(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := &Client{httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	})}}

	err := c.callMethod(ctx, sendTextMethod, testToken, []byte("{}"), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled in chain, got %v", err)
	}

	if strings.Contains(err.Error(), "test-token-telegram") {
		t.Fatalf("token leaked: %q", err.Error())
	}
}

func TestRedactToken(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"no token", errors.New("boom"), "boom"},
		{"plain text", errors.New("parse https://api.telegram.org/bot" + testToken + "/x: bad"), "parse https://api.telegram.org/bot<redacted>/x: bad"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactToken(tc.err, testToken)
			if tc.err == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}

				return
			}

			if got.Error() != tc.want {
				t.Fatalf("redactToken() = %q, want %q", got.Error(), tc.want)
			}
		})
	}
}
