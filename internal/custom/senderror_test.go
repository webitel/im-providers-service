package custom

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	kiterrors "github.com/webitel/webitel-go-kit/pkg/errors"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	custommodel "github.com/webitel/im-providers-service/internal/custom/model"
)

func TestSendErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"rejected", kiterrors.Append(custommodel.ErrCallbackRejected, "unknown chat"), sharedmodel.SendErrSendFailed},
		{"unreachable", kiterrors.Append(custommodel.ErrCallbackFailed, "dial tcp: refused"), sharedmodel.SendErrChannelUnavailable},
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

func TestUndeliveredCode(t *testing.T) {
	tooLarge := sharedmodel.NewSendError(sharedmodel.SendErrFileTooLarge,
		kiterrors.Append(custommodel.ErrCallbackFailed, "status 413 Request Entity Too Large"))

	cases := []struct {
		name  string
		cause error
		want  string
	}{
		{"attempts exhausted", kiterrors.Append(custommodel.ErrCallbackFailed, "status 502 Bad Gateway"), sharedmodel.SendErrChannelUnavailable},
		{"payload too large", tooLarge, sharedmodel.SendErrFileTooLarge},
		{"rejected on retry", kiterrors.Append(custommodel.ErrCallbackRejected, "unknown chat"), sharedmodel.SendErrSendFailed},
		{"no cause", nil, sharedmodel.SendErrChannelUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := undeliveredCode(tc.cause); got != tc.want {
				t.Errorf("undeliveredCode(%v) = %q, want %q", tc.cause, got, tc.want)
			}
		})
	}
}

func TestPost_HTTPStatusCodes(t *testing.T) {
	cases := []struct {
		name   string
		status int
		reply  string
		want   string
	}{
		{"413", http.StatusRequestEntityTooLarge, "", sharedmodel.SendErrFileTooLarge},
		{"502", http.StatusBadGateway, "", sharedmodel.SendErrChannelUnavailable},
		{"503 html", http.StatusServiceUnavailable, "<html>down</html>", sharedmodel.SendErrChannelUnavailable},
		{"408", http.StatusRequestTimeout, "", sharedmodel.SendErrChannelUnavailable},
		{"429", http.StatusTooManyRequests, "", sharedmodel.SendErrChannelUnavailable},
		{"400", http.StatusBadRequest, "", sharedmodel.SendErrSendFailed},
		{"401", http.StatusUnauthorized, "", sharedmodel.SendErrSendFailed},
		{"404", http.StatusNotFound, `{"success":false,"error":"no such chat"}`, sharedmodel.SendErrSendFailed},
		{"400 naming a normalized code", http.StatusBadRequest, `{"success":false,"error":"file_too_large"}`, sharedmodel.SendErrFileTooLarge},
		{"500 naming a normalized code", http.StatusInternalServerError, `{"error":"recipient_unavailable"}`, sharedmodel.SendErrRecipientUnavailable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, store, server := outboundFixture(t, knownChat())

			server.mu.Lock()
			server.status = tc.status
			server.reply = tc.reply
			server.mu.Unlock()

			err := p.api.post(t.Context(), store.gate, []byte(`{}`))
			if !errors.Is(err, custommodel.ErrCallbackFailed) {
				t.Fatalf("a non-2xx answer must stay retryable, got %v", err)
			}

			if got := sharedmodel.ClassifySendError(err); got != tc.want {
				t.Errorf("classified as %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSendText_RefusalCarriesSendFailed(t *testing.T) {
	p, _, server := outboundFixture(t, knownChat())
	server.refuse("unknown chat")

	_, err := p.SendText(t.Context(), operatorMessage())

	if got := sharedmodel.ClassifySendError(err); got != sharedmodel.SendErrSendFailed {
		t.Errorf("classified as %q", got)
	}
}

func TestPost_RefusalNamingNormalizedCode(t *testing.T) {
	p, store, server := outboundFixture(t, knownChat())
	server.refuse(sharedmodel.SendErrUnsupportedMedia)

	err := p.api.post(t.Context(), store.gate, []byte(`{}`))
	if !errors.Is(err, custommodel.ErrCallbackRejected) {
		t.Fatalf("a refusal must stay final, got %v", err)
	}

	if got := sharedmodel.ClassifySendError(err); got != sharedmodel.SendErrUnsupportedMedia {
		t.Errorf("classified as %q", got)
	}
}

func TestPost_TransportErrorRedactsCallbackCredentials(t *testing.T) {
	p, store, server := outboundFixture(t, knownChat())
	server.Close()

	gate := *store.gate
	gate.CallbackURL = strings.Replace(server.URL, "http://", "http://partner:test-token-basic@", 1) + "/hook?token=test-token-query"

	err := p.api.post(t.Context(), &gate, []byte(`{}`))
	if err == nil {
		t.Fatal("expected a transport error")
	}

	if strings.Contains(err.Error(), "test-token-") {
		t.Fatalf("credentials leaked: %q", err.Error())
	}

	if got := sharedmodel.ClassifySendError(tagSendError(err)); got != sharedmodel.SendErrChannelUnavailable {
		t.Errorf("classified as %q", got)
	}
}
