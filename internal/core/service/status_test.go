package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	threadv1 "github.com/webitel/im-providers-service/gen/go/thread/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

type fakeThreadClient struct {
	delivered []*threadv1.MarkDeliveredRequest
	read      []*threadv1.MarkReadRequest
	failed    []*threadv1.MarkFailedRequest
}

func (f *fakeThreadClient) MarkDelivered(_ context.Context, in *threadv1.MarkDeliveredRequest, _ ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	f.delivered = append(f.delivered, in)

	return &threadv1.MarkStatusResponse{}, nil
}

func (f *fakeThreadClient) MarkRead(_ context.Context, in *threadv1.MarkReadRequest, _ ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	f.read = append(f.read, in)

	return &threadv1.MarkStatusResponse{}, nil
}

func (f *fakeThreadClient) MarkFailed(_ context.Context, in *threadv1.MarkFailedRequest, _ ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	f.failed = append(f.failed, in)

	return &threadv1.MarkStatusResponse{}, nil
}

type fakeRefStore struct {
	saved []*sharedmodel.MessageRef

	refs    []*sharedmodel.MessageRef
	err     error
	gotIDs  []string
	gotUser string
	gotLim  int
}

func (f *fakeRefStore) Save(_ context.Context, ref *sharedmodel.MessageRef) error {
	f.saved = append(f.saved, ref)

	return f.err
}

func (f *fakeRefStore) GetByProviderMessageIDs(_ context.Context, _ string, ids []string) ([]*sharedmodel.MessageRef, error) {
	f.gotIDs = ids

	return f.refs, f.err
}

func (f *fakeRefStore) GetByUserUpTo(_ context.Context, _, providerUserID string, _ time.Time, limit int) ([]*sharedmodel.MessageRef, error) {
	f.gotUser = providerUserID
	f.gotLim = limit

	return f.refs, f.err
}

func newTestReporter(refs *fakeRefStore, thread *fakeThreadClient) *StatusReporter {
	return NewStatusReporter(slog.New(slog.NewTextHandler(io.Discard, nil)), thread, refs)
}

func ref(providerMessageID string) *sharedmodel.MessageRef {
	return &sharedmodel.MessageRef{
		GateID:            "gate-1",
		ProviderMessageID: providerMessageID,
		ProviderUserID:    "psid-1",
		MessageID:         uuid.New(),
		ThreadID:          uuid.New(),
		MemberID:          uuid.New(),
		DomainID:          7,
	}
}

func TestSaveRef_SkipsIncompleteRefs(t *testing.T) {
	cases := []struct {
		name string
		in   *sharedmodel.MessageRef
	}{
		{"nil ref", nil},
		{"empty provider message id", &sharedmodel.MessageRef{MessageID: uuid.New()}},
		{"nil message id", &sharedmodel.MessageRef{ProviderMessageID: "mid.1"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refs := &fakeRefStore{}
			newTestReporter(refs, &fakeThreadClient{}).SaveRef(context.Background(), tc.in)

			if len(refs.saved) != 0 {
				t.Fatalf("expected no save, got %d", len(refs.saved))
			}
		})
	}
}

func TestSaveRef_PersistsValidRef(t *testing.T) {
	refs := &fakeRefStore{}
	in := ref("mid.1")

	newTestReporter(refs, &fakeThreadClient{}).SaveRef(context.Background(), in)

	if len(refs.saved) != 1 || refs.saved[0] != in {
		t.Fatalf("expected the ref to be saved, got %+v", refs.saved)
	}
}

func TestSendFailure_InvalidIDsDisableTracking(t *testing.T) {
	valid := uuid.New().String()

	cases := []struct {
		name string
		in   SendFailureReport
	}{
		{"bad message id", SendFailureReport{MessageID: "nope", ThreadID: valid, MemberID: valid}},
		{"bad thread id", SendFailureReport{MessageID: valid, ThreadID: "", MemberID: valid}},
		{"bad member id", SendFailureReport{MessageID: valid, ThreadID: valid, MemberID: "x"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			thread := &fakeThreadClient{}
			newTestReporter(&fakeRefStore{}, thread).SendFailure(context.Background(), tc.in)

			if len(thread.failed) != 0 {
				t.Fatalf("expected no MarkFailed call, got %d", len(thread.failed))
			}
		})
	}
}

func TestSendFailure_ReportsFailureReceipt(t *testing.T) {
	thread := &fakeThreadClient{}
	in := SendFailureReport{
		MessageID:    uuid.New().String(),
		ThreadID:     uuid.New().String(),
		MemberID:     uuid.New().String(),
		DomainID:     3,
		ErrorCode:    "InvalidArgument",
		ErrorMessage: "recipient rejected",
	}

	newTestReporter(&fakeRefStore{}, thread).SendFailure(context.Background(), in)

	if len(thread.failed) != 1 || len(thread.failed[0].Receipts) != 1 {
		t.Fatalf("expected one MarkFailed receipt, got %+v", thread.failed)
	}

	got := thread.failed[0].Receipts[0]

	if got.MessageId != in.MessageID || got.ThreadId != in.ThreadID || got.MemberId != in.MemberID {
		t.Errorf("receipt ids mismatch: %+v", got)
	}

	if got.Via != statusVia || got.DomainId != in.DomainID {
		t.Errorf("via/domain mismatch: via=%q domain=%d", got.Via, got.DomainId)
	}

	if got.ErrorCode != in.ErrorCode || got.ErrorMessage != in.ErrorMessage {
		t.Errorf("error details mismatch: %+v", got)
	}

	if got.FailedAt <= 0 {
		t.Errorf("expected failed_at to be stamped, got %d", got.FailedAt)
	}
}

func TestDeliveredByProviderIDs_MapsRefsToReceipts(t *testing.T) {
	r1, r2 := ref("mid.1"), ref("mid.2")
	refs := &fakeRefStore{refs: []*sharedmodel.MessageRef{r1, r2}}
	thread := &fakeThreadClient{}
	at := time.UnixMilli(1_770_000_000_000)

	newTestReporter(refs, thread).DeliveredByProviderIDs(context.Background(), "gate-1", []string{"mid.1", "mid.2"}, at)

	if len(thread.delivered) != 1 || len(thread.delivered[0].Receipts) != 2 {
		t.Fatalf("expected one MarkDelivered call with 2 receipts, got %+v", thread.delivered)
	}

	got := thread.delivered[0].Receipts[0]

	if got.MessageId != r1.MessageID.String() || got.MemberId != r1.MemberID.String() || got.ThreadId != r1.ThreadID.String() {
		t.Errorf("receipt ids mismatch: %+v", got)
	}

	if got.DeliveredAt != at.UnixMilli() || got.Via != statusVia || got.DomainId != int32(r1.DomainID) {
		t.Errorf("receipt attrs mismatch: %+v", got)
	}
}

func TestDeliveredByProviderIDs_NoRefsOrError_NoReport(t *testing.T) {
	cases := []struct {
		name string
		refs *fakeRefStore
	}{
		{"store error", &fakeRefStore{err: errors.New("boom")}},
		{"no refs resolved", &fakeRefStore{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			thread := &fakeThreadClient{}
			newTestReporter(tc.refs, thread).DeliveredByProviderIDs(context.Background(), "gate-1", []string{"mid.1"}, time.Now())

			if len(thread.delivered) != 0 {
				t.Fatalf("expected no MarkDelivered call, got %d", len(thread.delivered))
			}
		})
	}
}

func TestDeliveredUpTo_UsesWatermarkLimit(t *testing.T) {
	refs := &fakeRefStore{refs: []*sharedmodel.MessageRef{ref("mid.1")}}
	thread := &fakeThreadClient{}

	newTestReporter(refs, thread).DeliveredUpTo(context.Background(), "gate-1", "psid-1", time.Now())

	if refs.gotLim != watermarkDeliveryLimit {
		t.Errorf("expected limit %d, got %d", watermarkDeliveryLimit, refs.gotLim)
	}

	if refs.gotUser != "psid-1" {
		t.Errorf("expected psid-1, got %q", refs.gotUser)
	}

	if len(thread.delivered) != 1 {
		t.Fatalf("expected one MarkDelivered call, got %d", len(thread.delivered))
	}
}

func TestReadUpTo_ReportsSingleReadReceipt(t *testing.T) {
	r := ref("mid.9")
	refs := &fakeRefStore{refs: []*sharedmodel.MessageRef{r}}
	thread := &fakeThreadClient{}
	watermark := time.UnixMilli(1_770_000_111_000)

	newTestReporter(refs, thread).ReadUpTo(context.Background(), "gate-1", "psid-1", watermark)

	if refs.gotLim != 1 {
		t.Errorf("read watermark must resolve a single ref, limit=%d", refs.gotLim)
	}

	if len(thread.read) != 1 || len(thread.read[0].Receipts) != 1 {
		t.Fatalf("expected one MarkRead receipt, got %+v", thread.read)
	}

	got := thread.read[0].Receipts[0]

	if got.UpToMessageId != r.MessageID.String() {
		t.Errorf("expected up_to_message_id %s, got %s", r.MessageID, got.UpToMessageId)
	}

	if got.ReadAt != watermark.UnixMilli() || got.Via != statusVia {
		t.Errorf("receipt attrs mismatch: %+v", got)
	}
}

func TestReadUpTo_NoRefs_NoReport(t *testing.T) {
	thread := &fakeThreadClient{}
	newTestReporter(&fakeRefStore{}, thread).ReadUpTo(context.Background(), "gate-1", "psid-1", time.Now())

	if len(thread.read) != 0 {
		t.Fatalf("expected no MarkRead call, got %d", len(thread.read))
	}
}

func TestReadByProviderID_ReportsReadReceipt(t *testing.T) {
	r := ref("wamid.1")
	refs := &fakeRefStore{refs: []*sharedmodel.MessageRef{r}}
	thread := &fakeThreadClient{}

	newTestReporter(refs, thread).ReadByProviderID(context.Background(), "gate-1", "wamid.1", time.Now())

	if len(thread.read) != 1 || thread.read[0].Receipts[0].UpToMessageId != r.MessageID.String() {
		t.Fatalf("expected read receipt for %s, got %+v", r.MessageID, thread.read)
	}
}

func TestFailedByProviderID_ReportsErrorDetails(t *testing.T) {
	r := ref("wamid.2")
	refs := &fakeRefStore{refs: []*sharedmodel.MessageRef{r}}
	thread := &fakeThreadClient{}

	newTestReporter(refs, thread).FailedByProviderID(context.Background(), "gate-1", "wamid.2", time.Now(), "131047", "re-engagement window expired")

	if len(thread.failed) != 1 || len(thread.failed[0].Receipts) != 1 {
		t.Fatalf("expected one MarkFailed receipt, got %+v", thread.failed)
	}

	got := thread.failed[0].Receipts[0]

	if got.ErrorCode != "131047" || got.ErrorMessage != "re-engagement window expired" {
		t.Errorf("error details mismatch: %+v", got)
	}

	if got.MessageId != r.MessageID.String() {
		t.Errorf("expected message id %s, got %s", r.MessageID, got.MessageId)
	}
}

func TestFailedByProviderID_NoRefs_NoReport(t *testing.T) {
	thread := &fakeThreadClient{}
	newTestReporter(&fakeRefStore{}, thread).FailedByProviderID(context.Background(), "gate-1", "wamid.3", time.Now(), "1", "boom")

	if len(thread.failed) != 0 {
		t.Fatalf("expected no MarkFailed call, got %d", len(thread.failed))
	}
}
