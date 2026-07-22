package webhook

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/webitel/im-providers-service/internal/whatsapp/common"
)

type fakeAccountResolver struct {
	account *common.WhatsappBusinessAccount
	err     error
}

func (f *fakeAccountResolver) Resolve(context.Context, *WhatsAppBusinessAccountResolveQuery) (*common.WhatsappBusinessAccount, error) {
	return f.account, f.err
}

// identityEncryptor satisfies common.Encryptor for the account PostFetch step.
type identityEncryptor struct{}

func (identityEncryptor) Encrypt(plaintext string) (string, error)  { return plaintext, nil }
func (identityEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type statusCall struct {
	kind, id, code, message string
	at                      time.Time
}

type fakeWAReporter struct {
	calls []statusCall
}

func (f *fakeWAReporter) DeliveredByProviderIDs(_ context.Context, _ string, ids []string, at time.Time) {
	for _, id := range ids {
		f.calls = append(f.calls, statusCall{kind: "delivered", id: id, at: at})
	}
}

func (f *fakeWAReporter) ReadByProviderID(_ context.Context, _, providerMessageID string, at time.Time) {
	f.calls = append(f.calls, statusCall{kind: "read", id: providerMessageID, at: at})
}

func (f *fakeWAReporter) FailedByProviderID(_ context.Context, _, providerMessageID string, at time.Time, code, message string) {
	f.calls = append(f.calls, statusCall{kind: "failed", id: providerMessageID, at: at, code: code, message: message})
}

func newStatusWebhook(resolver WhatsAppBusinessAccountResolver, reporter StatusReporter) *webhook {
	return newWebhook(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, resolver, identityEncryptor{}, nil, reporter)
}

func TestHandleStatuses_RoutesByStatusKind(t *testing.T) {
	reporter := &fakeWAReporter{}
	resolver := &fakeAccountResolver{account: &common.WhatsappBusinessAccount{ID: uuid.New()}}

	failedStatus := Status{ID: "wamid.f", Status: "failed", Timestamp: "1770000300"}
	failedStatus.Errors = []Error{{Code: 131047, Message: "Re-engagement message"}}

	err := newStatusWebhook(resolver, reporter).HandleStatuses(context.Background(), []Status{
		{ID: "wamid.d", Status: "delivered", Timestamp: "1770000100"},
		{ID: "wamid.r", Status: "read", Timestamp: "1770000200"},
		failedStatus,
		{ID: "wamid.s", Status: "sent", Timestamp: "1770000400"}, // initial SENT is created by im-thread
		{ID: "", Status: "delivered"},                            // no provider message id — skipped
	}, "phone-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reporter.calls) != 3 {
		t.Fatalf("expected 3 reports (sent and empty-id skipped), got %+v", reporter.calls)
	}

	if reporter.calls[0].kind != "delivered" || reporter.calls[0].id != "wamid.d" {
		t.Errorf("delivered mismatch: %+v", reporter.calls[0])
	}

	if !reporter.calls[0].at.Equal(time.Unix(1770000100, 0)) {
		t.Errorf("expected webhook timestamp, got %v", reporter.calls[0].at)
	}

	if reporter.calls[1].kind != "read" || reporter.calls[1].id != "wamid.r" {
		t.Errorf("read mismatch: %+v", reporter.calls[1])
	}

	failed := reporter.calls[2]
	if failed.kind != "failed" || failed.id != "wamid.f" || failed.code != "131047" || failed.message != "Re-engagement message" {
		t.Errorf("failed mismatch: %+v", failed)
	}
}

func TestHandleStatuses_DisabledAccount_NoReports(t *testing.T) {
	reporter := &fakeWAReporter{}
	resolver := &fakeAccountResolver{err: WebhookErrDisablled}

	err := newStatusWebhook(resolver, reporter).HandleStatuses(context.Background(), []Status{
		{ID: "wamid.d", Status: "delivered"},
	}, "phone-disabled")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reporter.calls) != 0 {
		t.Fatalf("expected no reports for a disabled account, got %+v", reporter.calls)
	}
}

func TestHandleStatuses_EmptyBatch_NoResolve(t *testing.T) {
	reporter := &fakeWAReporter{}

	// A nil resolver would panic if HandleStatuses touched it.
	err := newStatusWebhook(nil, reporter).HandleStatuses(context.Background(), nil, "phone-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(reporter.calls) != 0 {
		t.Fatalf("expected no reports, got %+v", reporter.calls)
	}
}

func TestStatusTimestamp(t *testing.T) {
	if got := statusTimestamp("1770000100"); !got.Equal(time.Unix(1770000100, 0)) {
		t.Errorf("expected parsed timestamp, got %v", got)
	}

	before := time.Now()
	for _, raw := range []string{"", "abc", "0", "-5"} {
		got := statusTimestamp(raw)
		if got.Before(before) || got.After(time.Now()) {
			t.Errorf("statusTimestamp(%q): expected ~now, got %v", raw, got)
		}
	}
}

func TestFirstStatusError(t *testing.T) {
	code, message := firstStatusError(nil)
	if code != "" || message != "" {
		t.Errorf("empty errors: got %q/%q", code, message)
	}

	plain := Error{Code: 470, Message: "message undeliverable"}

	code, message = firstStatusError([]Error{plain})
	if code != "470" || message != "message undeliverable" {
		t.Errorf("plain error: got %q/%q", code, message)
	}

	detailed := plain
	detailed.ErrorData.Details = "user has not opted in"

	code, message = firstStatusError([]Error{detailed, plain})
	if code != "470" || message != "user has not opted in" {
		t.Errorf("details must override message: got %q/%q", code, message)
	}
}
