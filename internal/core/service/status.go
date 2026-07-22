package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	threadv1 "github.com/webitel/im-providers-service/gen/go/thread/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
)

// ThreadStatusClient is the im-thread MessageStatus surface the reporter
// needs; satisfied by infra/client/grpc/im-thread.Client.
type ThreadStatusClient interface {
	MarkDelivered(ctx context.Context, in *threadv1.MarkDeliveredRequest, opts ...grpc.CallOption) (*threadv1.MarkStatusResponse, error)
	MarkRead(ctx context.Context, in *threadv1.MarkReadRequest, opts ...grpc.CallOption) (*threadv1.MarkStatusResponse, error)
	MarkFailed(ctx context.Context, in *threadv1.MarkFailedRequest, opts ...grpc.CallOption) (*threadv1.MarkStatusResponse, error)
}

// statusVia is the confirmation source reported to im-thread-service.
const statusVia = "provider"

// watermarkDeliveryLimit bounds how many recent refs a single watermark
// delivery receipt may confirm; earlier messages are covered by the
// monotonic upsert of later receipts.
const watermarkDeliveryLimit = 100

// StatusReporter resolves provider webhook receipts and synchronous send
// failures into MessageStatus reports for im-thread-service. All methods
// are best-effort: failures are logged, never propagated, so a receipt
// processing error cannot fail a provider webhook.
type StatusReporter struct {
	logger *slog.Logger
	thread ThreadStatusClient
	refs   sharedstore.MessageRefStore
}

func NewStatusReporter(logger *slog.Logger, thread ThreadStatusClient, refs sharedstore.MessageRefStore) *StatusReporter {
	return &StatusReporter{
		logger: logger.With(slog.String("component", "status_reporter")),
		thread: thread,
		refs:   refs,
	}
}

// SaveRef persists the provider_message_id mapping after a successful send.
func (s *StatusReporter) SaveRef(ctx context.Context, ref *sharedmodel.MessageRef) {
	if ref == nil || ref.ProviderMessageID == "" || ref.MessageID == uuid.Nil {
		return
	}

	if err := s.refs.Save(ctx, ref); err != nil {
		s.logger.ErrorContext(ctx, "failed to save message ref",
			"error", err,
			"gate_id", ref.GateID,
			"provider_message_id", ref.ProviderMessageID,
		)
	}
}

// SendFailure reports a synchronous provider send error as FAILED.
// The ids arrive as strings straight from the gRPC request; missing or
// invalid message context disables tracking silently (older callers).
func (s *StatusReporter) SendFailure(ctx context.Context, in SendFailureReport) {
	messageID, err := uuid.Parse(in.MessageID)
	if err != nil {
		return
	}

	threadID, err := uuid.Parse(in.ThreadID)
	if err != nil {
		return
	}

	memberID, err := uuid.Parse(in.MemberID)
	if err != nil {
		return
	}

	receipt := &threadv1.FailureReceipt{
		ThreadId:     threadID.String(),
		MessageId:    messageID.String(),
		MemberId:     memberID.String(),
		FailedAt:     time.Now().UnixMilli(),
		Via:          statusVia,
		DomainId:     in.DomainID,
		ErrorCode:    in.ErrorCode,
		ErrorMessage: in.ErrorMessage,
	}

	if _, err := s.thread.MarkFailed(ctx, &threadv1.MarkFailedRequest{
		Receipts: []*threadv1.FailureReceipt{receipt},
	}); err != nil {
		s.logger.ErrorContext(ctx, "failed to report send failure", "error", err, "message_id", in.MessageID)
	}
}

// SendFailureReport carries the context of a synchronous send error.
type SendFailureReport struct {
	MessageID    string
	ThreadID     string
	MemberID     string
	DomainID     int32
	ErrorCode    string
	ErrorMessage string
}

// DeliveredByProviderIDs reports delivery receipts referencing concrete
// provider message ids (WhatsApp statuses, Facebook delivery mids).
func (s *StatusReporter) DeliveredByProviderIDs(ctx context.Context, gateID string, providerMessageIDs []string, at time.Time) {
	refs, err := s.refs.GetByProviderMessageIDs(ctx, gateID, providerMessageIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve delivery refs", "error", err, "gate_id", gateID)

		return
	}

	s.markDelivered(ctx, refs, at)
}

// DeliveredUpTo reports a watermark delivery receipt: every message sent to
// the platform user before the watermark is confirmed as delivered.
func (s *StatusReporter) DeliveredUpTo(ctx context.Context, gateID, providerUserID string, watermark time.Time) {
	refs, err := s.refs.GetByUserUpTo(ctx, gateID, providerUserID, watermark, watermarkDeliveryLimit)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve watermark delivery refs", "error", err, "gate_id", gateID)

		return
	}

	s.markDelivered(ctx, refs, watermark)
}

func (s *StatusReporter) markDelivered(ctx context.Context, refs []*sharedmodel.MessageRef, at time.Time) {
	if len(refs) == 0 {
		return
	}

	receipts := make([]*threadv1.DeliveryReceipt, 0, len(refs))
	for _, ref := range refs {
		receipts = append(receipts, &threadv1.DeliveryReceipt{
			ThreadId:    ref.ThreadID.String(),
			MessageId:   ref.MessageID.String(),
			MemberId:    ref.MemberID.String(),
			DeliveredAt: at.UnixMilli(),
			Via:         statusVia,
			DomainId:    int32(ref.DomainID),
		})
	}

	if _, err := s.thread.MarkDelivered(ctx, &threadv1.MarkDeliveredRequest{Receipts: receipts}); err != nil {
		s.logger.ErrorContext(ctx, "failed to report delivered", "error", err, "receipts", len(receipts))
	}
}

// ReadUpTo reports a watermark read receipt (Facebook/Viber "seen"): the
// latest message sent to the user before the watermark closes all earlier
// ones through the read-up-to semantics of the thread service.
func (s *StatusReporter) ReadUpTo(ctx context.Context, gateID, providerUserID string, watermark time.Time) {
	refs, err := s.refs.GetByUserUpTo(ctx, gateID, providerUserID, watermark, 1)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve watermark read ref", "error", err, "gate_id", gateID)

		return
	}

	if len(refs) == 0 {
		return
	}

	s.markRead(ctx, refs[0], watermark)
}

// ReadByProviderID reports a per-message read receipt (WhatsApp): the
// referenced message closes all earlier ones through read-up-to semantics.
func (s *StatusReporter) ReadByProviderID(ctx context.Context, gateID, providerMessageID string, at time.Time) {
	refs, err := s.refs.GetByProviderMessageIDs(ctx, gateID, []string{providerMessageID})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve read ref", "error", err, "gate_id", gateID)

		return
	}

	if len(refs) == 0 {
		return
	}

	s.markRead(ctx, refs[0], at)
}

func (s *StatusReporter) markRead(ctx context.Context, ref *sharedmodel.MessageRef, at time.Time) {
	receipt := &threadv1.ReadReceipt{
		ThreadId:      ref.ThreadID.String(),
		MemberId:      ref.MemberID.String(),
		UpToMessageId: ref.MessageID.String(),
		ReadAt:        at.UnixMilli(),
		Via:           statusVia,
		DomainId:      int32(ref.DomainID),
	}

	if _, err := s.thread.MarkRead(ctx, &threadv1.MarkReadRequest{
		Receipts: []*threadv1.ReadReceipt{receipt},
	}); err != nil {
		s.logger.ErrorContext(ctx, "failed to report read", "error", err, "thread_id", receipt.ThreadId)
	}
}

// FailedByProviderID reports a failed webhook status referencing a concrete
// provider message id (WhatsApp "failed" status with error details).
func (s *StatusReporter) FailedByProviderID(ctx context.Context, gateID, providerMessageID string, at time.Time, errorCode, errorMessage string) {
	refs, err := s.refs.GetByProviderMessageIDs(ctx, gateID, []string{providerMessageID})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to resolve failure ref", "error", err, "gate_id", gateID)

		return
	}

	if len(refs) == 0 {
		return
	}

	ref := refs[0]
	receipt := &threadv1.FailureReceipt{
		ThreadId:     ref.ThreadID.String(),
		MessageId:    ref.MessageID.String(),
		MemberId:     ref.MemberID.String(),
		FailedAt:     at.UnixMilli(),
		Via:          statusVia,
		DomainId:     int32(ref.DomainID),
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}

	if _, err := s.thread.MarkFailed(ctx, &threadv1.MarkFailedRequest{
		Receipts: []*threadv1.FailureReceipt{receipt},
	}); err != nil {
		s.logger.ErrorContext(ctx, "failed to report failed status", "error", err, "thread_id", receipt.ThreadId)
	}
}
