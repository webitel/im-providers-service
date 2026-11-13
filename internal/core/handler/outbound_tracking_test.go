package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	threadv1 "github.com/webitel/im-providers-service/gen/go/thread/v1"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	coreservice "github.com/webitel/im-providers-service/internal/core/service"
	"github.com/webitel/im-providers-service/internal/provider"
)

type fakeThreadStatus struct {
	failed []*threadv1.FailureReceipt
}

func (f *fakeThreadStatus) MarkDelivered(context.Context, *threadv1.MarkDeliveredRequest, ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	return &threadv1.MarkStatusResponse{}, nil
}

func (f *fakeThreadStatus) MarkRead(context.Context, *threadv1.MarkReadRequest, ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	return &threadv1.MarkStatusResponse{}, nil
}

func (f *fakeThreadStatus) MarkFailed(_ context.Context, in *threadv1.MarkFailedRequest, _ ...grpc.CallOption) (*threadv1.MarkStatusResponse, error) {
	f.failed = append(f.failed, in.GetReceipts()...)

	return &threadv1.MarkStatusResponse{}, nil
}

type fakeRefs struct {
	saved []*sharedmodel.MessageRef
}

func (f *fakeRefs) Save(_ context.Context, ref *sharedmodel.MessageRef) error {
	f.saved = append(f.saved, ref)

	return nil
}

func (f *fakeRefs) GetByProviderMessageIDs(context.Context, string, []string) ([]*sharedmodel.MessageRef, error) {
	return nil, nil
}

func (f *fakeRefs) GetByUserUpTo(context.Context, string, string, time.Time, int) ([]*sharedmodel.MessageRef, error) {
	return nil, nil
}

// fakeTypeStore resolves every gate to viber; only GetTypeByID is reached by
// the outbound handler.
type fakeTypeStore struct{}

func (fakeTypeStore) List(context.Context, sharedmodel.ListFilter) ([]*sharedmodel.GateSummary, bool, error) {
	return nil, false, nil
}

func (fakeTypeStore) Get(context.Context, int64, string) (*sharedmodel.GateSummary, error) {
	return &sharedmodel.GateSummary{}, nil
}

func (fakeTypeStore) Delete(context.Context, string) error { return nil }

func (fakeTypeStore) GetTypeByID(context.Context, string) (sharedmodel.GateType, error) {
	return sharedmodel.TypeViber, nil
}

type fakeSender struct {
	resp *sharedmodel.MessageResponse
	err  error
}

func (f *fakeSender) Type() string { return sharedmodel.TypeViber.String() }

func (f *fakeSender) HandleWebhook(context.Context, []byte) error { return nil }

func (f *fakeSender) send() (*sharedmodel.MessageResponse, error) { return f.resp, f.err }

func (f *fakeSender) SendText(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return f.send()
}

func (f *fakeSender) SendImage(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return f.send()
}

func (f *fakeSender) SendDocument(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return f.send()
}

func (f *fakeSender) SendLocation(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return f.send()
}

func (f *fakeSender) SendContact(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return f.send()
}

func newTrackingHandler(t *testing.T, sender *fakeSender) (*OutboundMessageHandler, *fakeThreadStatus, *fakeRefs) {
	t.Helper()

	thread, refs := &fakeThreadStatus{}, &fakeRefs{}

	h, err := NewOutboundMessageHandler(
		noopLogger,
		provider.NewRegistry([]provider.Provider{sender}),
		fakeTypeStore{},
		nil,
		coreservice.NewStatusReporter(noopLogger, thread, refs),
	)
	if err != nil {
		t.Fatalf("NewOutboundMessageHandler: %v", err)
	}

	return h, thread, refs
}

func TestTrackOutcome_ReportsNormalizedCode(t *testing.T) {
	providerErr := errors.New("viber: attachment 60000000 bytes exceeds the platform limit of 52428800")

	cases := []struct {
		name string
		err  error
		want string
	}{
		{"tagged channel error", fmt.Errorf("send: %w", sharedmodel.NewSendError(sharedmodel.SendErrFileTooLarge, providerErr)), sharedmodel.SendErrFileTooLarge},
		{"timeout", fmt.Errorf("post: %w", context.DeadlineExceeded), sharedmodel.SendErrChannelUnavailable},
		{"untagged", providerErr, sharedmodel.SendErrSendFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, thread, _ := newTrackingHandler(t, &fakeSender{})
			mc := messageContextOf(uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), 1)

			h.trackOutcome(context.Background(), mc, nil, tc.err)

			if len(thread.failed) != 1 {
				t.Fatalf("expected one failure receipt, got %d", len(thread.failed))
			}

			got := thread.failed[0]
			if got.GetErrorCode() != tc.want {
				t.Errorf("error_code = %q, want %q", got.GetErrorCode(), tc.want)
			}

			if got.GetErrorMessage() != tc.err.Error() {
				t.Errorf("error_message = %q, want the raw error %q", got.GetErrorMessage(), tc.err.Error())
			}
		})
	}
}

func TestSendLocationAndContact_TrackOutcome(t *testing.T) {
	failure := sharedmodel.NewSendError(sharedmodel.SendErrRecipientUnavailable, errors.New("receiver not subscribed"))

	type call func(h *OutboundMessageHandler, gateID, userID, messageID, threadID string) error

	calls := []struct {
		name string
		call call
	}{
		{"location", func(h *OutboundMessageHandler, gateID, userID, messageID, threadID string) error {
			_, err := h.SendLocation(context.Background(), &impb.ProviderSendLocationRequest{
				GateId: gateID, ExternalUserId: userID, MessageId: messageID, ThreadId: threadID, Latitude: 1, Longitude: 2,
			})

			return err
		}},
		{"contact", func(h *OutboundMessageHandler, gateID, userID, messageID, threadID string) error {
			_, err := h.SendContact(context.Background(), &impb.ProviderSendContactRequest{
				GateId: gateID, ExternalUserId: userID, MessageId: messageID, ThreadId: threadID, Name: "Test", PhoneNumber: "+380000000000",
			})

			return err
		}},
	}

	for _, c := range calls {
		t.Run(c.name+" failure", func(t *testing.T) {
			h, thread, _ := newTrackingHandler(t, &fakeSender{err: failure})
			messageID := uuid.NewString()

			if err := c.call(h, uuid.NewString(), uuid.NewString(), messageID, uuid.NewString()); err == nil {
				t.Fatal("expected the send error to surface")
			}

			if len(thread.failed) != 1 {
				t.Fatalf("expected one failure receipt, got %d", len(thread.failed))
			}

			if got := thread.failed[0]; got.GetMessageId() != messageID || got.GetErrorCode() != sharedmodel.SendErrRecipientUnavailable {
				t.Errorf("unexpected receipt: %+v", got)
			}
		})

		t.Run(c.name+" success", func(t *testing.T) {
			h, thread, refs := newTrackingHandler(t, &fakeSender{resp: &sharedmodel.MessageResponse{ID: "provider-msg-1"}})
			messageID := uuid.NewString()

			if err := c.call(h, uuid.NewString(), uuid.NewString(), messageID, uuid.NewString()); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(thread.failed) != 0 {
				t.Errorf("expected no failure receipt, got %d", len(thread.failed))
			}

			if len(refs.saved) != 1 || refs.saved[0].MessageID.String() != messageID || refs.saved[0].ProviderMessageID != "provider-msg-1" {
				t.Errorf("expected the provider ref to be saved, got %+v", refs.saved)
			}
		})
	}
}

// plainSender implements only the base Sender surface: no interactive,
// location or contact support.
type plainSender struct{}

func (plainSender) Type() string                                { return sharedmodel.TypeViber.String() }
func (plainSender) HandleWebhook(context.Context, []byte) error { return nil }

func (plainSender) SendText(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return &sharedmodel.MessageResponse{}, nil
}

func (plainSender) SendImage(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return &sharedmodel.MessageResponse{}, nil
}

func (plainSender) SendDocument(context.Context, *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	return &sharedmodel.MessageResponse{}, nil
}

func TestEarlyRejections_CodesWithoutFailureReport(t *testing.T) {
	newHandler := func(t *testing.T, providers []provider.Provider) (*OutboundMessageHandler, *fakeThreadStatus) {
		t.Helper()

		thread := &fakeThreadStatus{}

		h, err := NewOutboundMessageHandler(noopLogger, provider.NewRegistry(providers), fakeTypeStore{}, nil,
			coreservice.NewStatusReporter(noopLogger, thread, &fakeRefs{}))
		if err != nil {
			t.Fatalf("NewOutboundMessageHandler: %v", err)
		}

		return h, thread
	}

	cases := []struct {
		name      string
		providers []provider.Provider
		call      func(h *OutboundMessageHandler) error
		want      codes.Code
	}{
		{"provider not registered", nil, func(h *OutboundMessageHandler) error {
			_, err := h.SendText(context.Background(), &impb.ProviderSendTextRequest{
				GateId: uuid.NewString(), ExternalUserId: uuid.NewString(), MessageId: uuid.NewString(), ThreadId: uuid.NewString(), Text: "hi",
			})

			return err
		}, codes.FailedPrecondition},
		{"location unsupported", []provider.Provider{plainSender{}}, func(h *OutboundMessageHandler) error {
			_, err := h.SendLocation(context.Background(), &impb.ProviderSendLocationRequest{
				GateId: uuid.NewString(), ExternalUserId: uuid.NewString(), MessageId: uuid.NewString(), ThreadId: uuid.NewString(),
			})

			return err
		}, codes.Unimplemented},
		{"malformed external user id", []provider.Provider{plainSender{}}, func(h *OutboundMessageHandler) error {
			_, err := h.SendText(context.Background(), &impb.ProviderSendTextRequest{
				GateId: uuid.NewString(), ExternalUserId: "not-a-uuid", MessageId: uuid.NewString(), ThreadId: uuid.NewString(), Text: "hi",
			})

			return err
		}, codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, thread := newHandler(t, tc.providers)

			if got := status.Code(tc.call(h)); got != tc.want {
				t.Errorf("code = %s, want %s", got, tc.want)
			}

			if len(thread.failed) != 0 {
				t.Errorf("early rejections are recorded by im-thread, got %d receipts here", len(thread.failed))
			}
		})
	}
}

func TestSendText_RedactsCredentialsInErrors(t *testing.T) {
	leaky := sharedmodel.NewSendError(sharedmodel.SendErrChannelUnavailable,
		status.Error(codes.Unavailable, `Post "https://api.telegram.org/bot123456:test-token-telegram/sendMessage": i/o timeout`))

	h, thread, _ := newTrackingHandler(t, &fakeSender{err: leaky})

	_, err := h.SendText(context.Background(), &impb.ProviderSendTextRequest{
		GateId: uuid.NewString(), ExternalUserId: uuid.NewString(), MessageId: uuid.NewString(), ThreadId: uuid.NewString(), Text: "hi",
	})

	if err == nil || strings.Contains(err.Error(), "test-token-telegram") {
		t.Fatalf("returned error leaks the token: %v", err)
	}

	if status.Code(err) != codes.Unavailable {
		t.Errorf("code = %s, want %s", status.Code(err), codes.Unavailable)
	}

	if len(thread.failed) != 1 || strings.Contains(thread.failed[0].GetErrorMessage(), "test-token-telegram") {
		t.Fatalf("failure receipt leaks the token: %+v", thread.failed)
	}
}
