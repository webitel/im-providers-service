package facebook

import (
	"context"
	"errors"
	"log/slog"
	"time"

	backoff "github.com/cenkalti/backoff/v3"
	"github.com/sony/gobreaker"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

// resilientGraphAPI wraps any graphAPI implementation with exponential-backoff retry
// (inner layer) and a circuit breaker (outer layer).
//
// Order: circuitBreaker → retry → apiClient
//
// The circuit breaker sees each full retry chain as a single attempt: it opens
// after 5 consecutive failed chains, preventing downstream hammering during outages.
// The retry window is capped at 30 s with 500 ms initial interval.
type resilientGraphAPI struct {
	inner      graphAPI
	breaker    *gobreaker.CircuitBreaker
	logger     *slog.Logger
	newBackOff func() backoff.BackOff
}

func newResilientGraphAPI(inner graphAPI, logger *slog.Logger) graphAPI {
	l := logger.With("component", "fb.api.resilient")
	settings := gobreaker.Settings{
		Name:    "fb-graph-api",
		Timeout: 60 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			l.Warn("circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	}
	return &resilientGraphAPI{
		inner:   inner,
		breaker: gobreaker.NewCircuitBreaker(settings),
		logger:  l,
		newBackOff: func() backoff.BackOff {
			b := backoff.NewExponentialBackOff()
			b.InitialInterval = 500 * time.Millisecond
			b.Multiplier = 2.0
			b.MaxInterval = 10 * time.Second
			b.MaxElapsedTime = 30 * time.Second
			return b
		},
	}
}

// retryable wraps err so that non-transient failures are not retried.
// ErrTokenInvalid (OAuth code 190), context errors, and backoff.Permanent
// errors short-circuit immediately; all other errors are retried until
// MaxElapsedTime.
func retryable(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTokenInvalid) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return backoff.Permanent(err)
	}
	return err
}

func (r *resilientGraphAPI) ParseWebhook(data []byte) (*WebhookRequest, error) {
	return r.inner.ParseWebhook(data)
}

func (r *resilientGraphAPI) GetUserProfile(ctx context.Context, psid, token string) (*UserProfile, error) {
	res, err := r.breaker.Execute(func() (any, error) {
		var profile *UserProfile
		err := backoff.Retry(func() error {
			var e error
			profile, e = r.inner.GetUserProfile(ctx, psid, token)
			return retryable(e)
		}, backoff.WithContext(r.newBackOff(), ctx))
		return profile, err
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(*UserProfile), nil
}

func (r *resilientGraphAPI) SendText(ctx context.Context, token, psid, text string) (*sharedmodel.MessageResponse, error) {
	res, err := r.breaker.Execute(func() (any, error) {
		var resp *sharedmodel.MessageResponse
		err := backoff.Retry(func() error {
			var e error
			resp, e = r.inner.SendText(ctx, token, psid, text)
			return retryable(e)
		}, backoff.WithContext(r.newBackOff(), ctx))
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(*sharedmodel.MessageResponse), nil
}

func (r *resilientGraphAPI) SendMedia(ctx context.Context, token, psid, mediaType, rawURL string) (*sharedmodel.MessageResponse, error) {
	res, err := r.breaker.Execute(func() (any, error) {
		var resp *sharedmodel.MessageResponse
		err := backoff.Retry(func() error {
			var e error
			resp, e = r.inner.SendMedia(ctx, token, psid, mediaType, rawURL)
			return retryable(e)
		}, backoff.WithContext(r.newBackOff(), ctx))
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(*sharedmodel.MessageResponse), nil
}

func (r *resilientGraphAPI) SendInteractive(ctx context.Context, token, psid, body string, interactive *sharedmodel.Interactive) (*sharedmodel.MessageResponse, error) {
	res, err := r.breaker.Execute(func() (any, error) {
		var resp *sharedmodel.MessageResponse
		err := backoff.Retry(func() error {
			var e error
			resp, e = r.inner.SendInteractive(ctx, token, psid, body, interactive)
			return retryable(e)
		}, backoff.WithContext(r.newBackOff(), ctx))
		return resp, err
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(*sharedmodel.MessageResponse), nil
}

func (r *resilientGraphAPI) SendTyping(ctx context.Context, token, psid string, on bool) error {
	_, err := r.breaker.Execute(func() (any, error) {
		return nil, backoff.Retry(func() error {
			return retryable(r.inner.SendTyping(ctx, token, psid, on))
		}, backoff.WithContext(r.newBackOff(), ctx))
	})

	return err
}

func (r *resilientGraphAPI) SetMessengerProfile(ctx context.Context, token string, profile any) error {
	_, err := r.breaker.Execute(func() (any, error) {
		return nil, backoff.Retry(func() error {
			return retryable(r.inner.SetMessengerProfile(ctx, token, profile))
		}, backoff.WithContext(r.newBackOff(), ctx))
	})
	return err
}

func (r *resilientGraphAPI) DeleteMessengerProfile(ctx context.Context, token string, fields []string) error {
	_, err := r.breaker.Execute(func() (any, error) {
		return nil, backoff.Retry(func() error {
			return retryable(r.inner.DeleteMessengerProfile(ctx, token, fields))
		}, backoff.WithContext(r.newBackOff(), ctx))
	})
	return err
}
