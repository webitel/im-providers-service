package facebook

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	backoff "github.com/cenkalti/backoff/v3"
	"github.com/sony/gobreaker"

	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
)

const profilePermissionBody = `{"error":{"message":"(#100) Insufficient permission to access user profile.","type":"OAuthException","code":100,"error_subcode":2018247}}`

// fakeProfileAPI implements only GetUserProfile; any other graphAPI call panics.
type fakeProfileAPI struct {
	graphAPI
	profile *UserProfile
	err     error
	calls   int
}

func (f *fakeProfileAPI) GetUserProfile(context.Context, string, string) (*UserProfile, error) {
	f.calls++
	return f.profile, f.err
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestGetUserProfileErrors(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		wantToken     bool
		wantPermanent bool
	}{
		{"permission denied is a permanent API error", http.StatusBadRequest, profilePermissionBody, false, true},
		{"rate limit is a transient API error", http.StatusTooManyRequests, `{"error":{"code":4}}`, false, false},
		{"server error is a transient API error", http.StatusInternalServerError, `{"error":{"code":2}}`, false, false},
		{"revoked token maps to ErrTokenInvalid", http.StatusBadRequest, `{"error":{"code":190}}`, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := &apiClient{client: srv.Client(), logger: discardLogger(), apiURL: srv.URL}

			_, err := c.GetUserProfile(context.Background(), "psid-1", "test-token-page")
			if err == nil {
				t.Fatal("expected an error")
			}

			if got := errors.Is(err, ErrTokenInvalid); got != tt.wantToken {
				t.Fatalf("ErrTokenInvalid = %v, want %v (err=%v)", got, tt.wantToken, err)
			}
			if tt.wantToken {
				return
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %T: %v", err, err)
			}
			if apiErr.StatusCode != tt.status || apiErr.Permanent() != tt.wantPermanent {
				t.Errorf("APIError status=%d permanent=%v, want %d/%v", apiErr.StatusCode, apiErr.Permanent(), tt.status, tt.wantPermanent)
			}
		})
	}
}

func TestBreakerSuccess(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, true},
		{"permission denied 400", &APIError{StatusCode: 400, Body: profilePermissionBody}, true},
		{"wrapped permanent API error", errors.Join(errors.New("fb profile"), &APIError{StatusCode: 403}), true},
		{"token invalid", ErrTokenInvalid, true},
		{"context canceled", context.Canceled, true},
		{"rate limit 429", &APIError{StatusCode: 429}, false},
		{"server error 500", &APIError{StatusCode: 500}, false},
		{"deadline exceeded", context.DeadlineExceeded, false},
		{"network error", errors.New("connection reset"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := breakerSuccess(tt.err); got != tt.want {
				t.Errorf("breakerSuccess(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func newTestResilient(inner graphAPI) *resilientGraphAPI {
	r := newResilientGraphAPI(inner, discardLogger()).(*resilientGraphAPI)
	r.newBackOff = func() backoff.BackOff { return &backoff.StopBackOff{} }
	return r
}

// A burst of permission errors for users without an app role must not open
// the breaker and block Graph API for every other user and gate.
func TestResilientBreakerIgnoresPermanentErrors(t *testing.T) {
	inner := &fakeProfileAPI{err: &APIError{StatusCode: 400, Body: profilePermissionBody}}
	r := newTestResilient(inner)

	for range 10 {
		_, _ = r.GetUserProfile(context.Background(), "psid-1", "test-token-page")
	}

	if st := r.breaker.State(); st != gobreaker.StateClosed {
		t.Fatalf("breaker state = %v, want closed", st)
	}
	if inner.calls != 10 {
		t.Errorf("inner calls = %d, want 10 (one per request, no retries for a permanent error)", inner.calls)
	}
}

func TestResilientBreakerOpensOnTransientErrors(t *testing.T) {
	inner := &fakeProfileAPI{err: &APIError{StatusCode: 500}}
	r := newTestResilient(inner)

	for range 5 {
		_, _ = r.GetUserProfile(context.Background(), "psid-1", "test-token-page")
	}

	if st := r.breaker.State(); st != gobreaker.StateOpen {
		t.Fatalf("breaker state = %v, want open", st)
	}
	if _, err := r.GetUserProfile(context.Background(), "psid-1", "test-token-page"); !errors.Is(err, gobreaker.ErrOpenState) {
		t.Errorf("expected ErrOpenState while open, got %v", err)
	}
}

func TestUserProfileFallsBackToPSID(t *testing.T) {
	gate := &fbmodel.FacebookGate{ID: "gate-1", PageToken: "test-token-page"}

	tests := []struct {
		name string
		api  *fakeProfileAPI
		want UserProfile
	}{
		{
			name: "profile available",
			api:  &fakeProfileAPI{profile: &UserProfile{ID: "psid-1", FirstName: "Test", LastName: "User"}},
			want: UserProfile{ID: "psid-1", FirstName: "Test", LastName: "User"},
		},
		{
			name: "permission denied",
			api:  &fakeProfileAPI{err: &APIError{StatusCode: 400, Body: profilePermissionBody}},
			want: UserProfile{ID: "psid-1"},
		},
		{
			name: "breaker open",
			api:  &fakeProfileAPI{err: gobreaker.ErrOpenState},
			want: UserProfile{ID: "psid-1"},
		},
		{
			name: "nil profile without error",
			api:  &fakeProfileAPI{},
			want: UserProfile{ID: "psid-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &facebookProvider{api: tt.api, logger: discardLogger()}

			got := p.userProfile(context.Background(), gate, "psid-1")
			if got == nil || *got != tt.want {
				t.Errorf("userProfile = %+v, want %+v", got, tt.want)
			}
		})
	}
}
