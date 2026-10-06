package webhook

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider"
)

type baseProvider struct {
	handled bool
}

func (p *baseProvider) Type() string { return "test" }

func (p *baseProvider) SendText(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return &sharedmodel.MessageResponse{}, nil
}

func (p *baseProvider) SendImage(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return &sharedmodel.MessageResponse{}, nil
}

func (p *baseProvider) SendDocument(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return &sharedmodel.MessageResponse{}, nil
}

func (p *baseProvider) HandleWebhook(context.Context, []byte) error {
	p.handled = true

	return nil
}

type stringSigProvider struct {
	baseProvider

	got string
}

func (p *stringSigProvider) ValidateSignature(_ context.Context, header string, _ []byte) error {
	p.got = header
	if header != "sha256=ok" {
		return errors.New("bad signature")
	}

	return nil
}

type headerSigProvider struct {
	baseProvider
}

func (p *headerSigProvider) ValidateSignature(_ context.Context, headers http.Header, _ []byte) error {
	if headers.Get("Authorization") != "App secret" {
		return errors.New("bad credential")
	}

	return nil
}

func serve(t *testing.T, p provider.Provider, headers map[string]string) int {
	t.Helper()

	h := NewHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), []provider.Provider{p})

	r := httptest.NewRequest(http.MethodPost, "/wh/test/abc", strings.NewReader("{}"))
	for k, v := range headers {
		r.Header.Set(k, v)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "test")
	rctx.URLParams.Add("uri", "abc")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	return w.Code
}

func TestServeHTTPSignatureDispatch(t *testing.T) {
	t.Run("string validator gets X-Hub-Signature-256", func(t *testing.T) {
		p := &stringSigProvider{}
		if code := serve(t, p, map[string]string{"X-Hub-Signature-256": "sha256=ok"}); code != http.StatusOK || !p.handled {
			t.Fatalf("code = %d, handled = %v", code, p.handled)
		}

		if p.got != "sha256=ok" {
			t.Errorf("header = %q", p.got)
		}
	})

	t.Run("string validator rejects", func(t *testing.T) {
		p := &stringSigProvider{}
		if code := serve(t, p, nil); code != http.StatusForbidden || p.handled {
			t.Fatalf("code = %d, handled = %v", code, p.handled)
		}
	})

	t.Run("header validator sees Authorization", func(t *testing.T) {
		p := &headerSigProvider{}
		if code := serve(t, p, map[string]string{"Authorization": "App secret"}); code != http.StatusOK || !p.handled {
			t.Fatalf("code = %d, handled = %v", code, p.handled)
		}
	})

	t.Run("header validator rejects", func(t *testing.T) {
		p := &headerSigProvider{}
		if code := serve(t, p, map[string]string{"Authorization": "App wrong"}); code != http.StatusForbidden || p.handled {
			t.Fatalf("code = %d, handled = %v", code, p.handled)
		}
	})

	t.Run("no validator passes", func(t *testing.T) {
		p := &baseProvider{}
		if code := serve(t, p, nil); code != http.StatusOK || !p.handled {
			t.Fatalf("code = %d, handled = %v", code, p.handled)
		}
	})
}
