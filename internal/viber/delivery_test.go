package viber

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
	vibstore "github.com/webitel/im-providers-service/internal/viber/store"
)

type fakeGateCache struct{}

func (fakeGateCache) Set(string, sharedstore.GateState)        {}
func (fakeGateCache) Get(string) (sharedstore.GateState, bool) { return sharedstore.GateState{}, false }
func (fakeGateCache) Delete(string)                            {}

type fakeViberStore struct {
	vibstore.ViberStore

	gate *vibmodel.ViberGate
}

func (f *fakeViberStore) SelectByURI(context.Context, string) (*vibmodel.ViberGate, error) {
	return f.gate, nil
}

type fakeMessenger struct {
	sharedsvc.Messenger

	reports []*sharedmodel.MessageDeliveryReport
}

func (f *fakeMessenger) UpdateMessageDelivery(_ context.Context, in *sharedmodel.MessageDeliveryReport) error {
	f.reports = append(f.reports, in)

	return nil
}

type failedCall struct {
	gateID, providerMessageID, code, message string
	at                                       time.Time
}

type fakeFailureReporter struct {
	calls []failedCall
}

func (f *fakeFailureReporter) FailedByProviderID(_ context.Context, gateID, providerMessageID string, at time.Time, errorCode, errorMessage string) {
	f.calls = append(f.calls, failedCall{gateID: gateID, providerMessageID: providerMessageID, code: errorCode, message: errorMessage, at: at})
}

func newDeliveryTestProvider() (*viberProvider, *fakeMessenger, *fakeFailureReporter) {
	messenger := &fakeMessenger{}
	failures := &fakeFailureReporter{}

	return &viberProvider{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		messenger: messenger,
		gateCache: fakeGateCache{},
		repo:      &fakeViberStore{gate: &vibmodel.ViberGate{ID: "gate-1", Enabled: true, DomainID: 1}},
		status:    failures,
	}, messenger, failures
}

func TestReportDelivery_FailedRecordsBothDeliveryAndFailure(t *testing.T) {
	p, messenger, failures := newDeliveryTestProvider()
	evt := &inboundEvent{
		Event:        "failed",
		MessageToken: "5741311803571721087",
		UserID:       "test-viber-user",
		Desc:         "file size exceeds the limit",
		Timestamp:    1_770_000_000_000,
	}

	if err := p.reportDelivery(context.Background(), evt, deliveryFailed); err != nil {
		t.Fatalf("reportDelivery: %v", err)
	}

	if len(failures.calls) != 1 {
		t.Fatalf("expected one failure report, got %d", len(failures.calls))
	}

	got := failures.calls[0]
	if got.gateID != "gate-1" || got.providerMessageID != "5741311803571721087" ||
		got.code != sharedmodel.SendErrFileTooLarge || got.message != evt.Desc || !got.at.Equal(time.UnixMilli(evt.Timestamp)) {
		t.Errorf("unexpected failure report: %+v", got)
	}

	if len(messenger.reports) != 1 {
		t.Fatalf("expected one delivery update, got %d", len(messenger.reports))
	}

	rep := messenger.reports[0]
	if rep.Status != deliveryFailed || rep.ExternalID != "5741311803571721087" || rep.Reason != evt.Desc || rep.GateID != "gate-1" {
		t.Errorf("unexpected delivery update: %+v", rep)
	}
}

func TestReportDelivery_DeliveredSkipsFailureReport(t *testing.T) {
	p, messenger, failures := newDeliveryTestProvider()
	evt := &inboundEvent{Event: "delivered", MessageToken: "5741311803571721088", UserID: "test-viber-user"}

	if err := p.reportDelivery(context.Background(), evt, deliveryDelivered); err != nil {
		t.Fatalf("reportDelivery: %v", err)
	}

	if len(failures.calls) != 0 {
		t.Errorf("expected no failure report, got %+v", failures.calls)
	}

	if len(messenger.reports) != 1 || messenger.reports[0].Status != deliveryDelivered {
		t.Errorf("expected one delivered update, got %+v", messenger.reports)
	}
}
