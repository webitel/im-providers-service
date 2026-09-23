package viberbm

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/provider"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

// fakeStatusReporter captures every statusReporter call for assertion.
type fakeStatusReporter struct {
	deliveredIDs    []string
	deliveredGateID string
	deliveredAt     time.Time

	readGate string
	readUpTo string
	readAt   time.Time

	failedGate    string
	failedID      string
	failedAt      time.Time
	failedCode    string
	failedMessage string
}

func (f *fakeStatusReporter) DeliveredByProviderIDs(_ context.Context, gateID string, ids []string, at time.Time) {
	f.deliveredGateID = gateID
	f.deliveredIDs = ids
	f.deliveredAt = at
}

func (f *fakeStatusReporter) ReadUpTo(_ context.Context, gateID, providerUserID string, at time.Time) {
	f.readGate = gateID
	f.readUpTo = providerUserID
	f.readAt = at
}

func (f *fakeStatusReporter) FailedByProviderID(_ context.Context, gateID, mid string, at time.Time, code, msg string) {
	f.failedGate = gateID
	f.failedID = mid
	f.failedAt = at
	f.failedCode = code
	f.failedMessage = msg
}

// fakeUserCache implements sharedstore.ExternalUserCache — always reports the
// user as already known so syncContact returns immediately without touching
// the gateway or Redis. This is the correct posture for unit tests that only
// care about the message-routing logic after contact resolution.
type fakeUserCache struct{}

func (fakeUserCache) IsKnown(_ context.Context, _ *sharedmodel.ExternalUser) (bool, error) {
	return true, nil
}

func (fakeUserCache) MarkKnown(_ context.Context, _ *sharedmodel.ExternalUser) error {
	return nil
}

// fakeMessenger implements sharedsvc.Messenger — records every call.
type fakeMessenger struct {
	textReqs     []*sharedmodel.SendTextRequest
	imageReqs    []*sharedmodel.SendImageRequest
	documentReqs []*sharedmodel.SendDocumentRequest
}

func (m *fakeMessenger) SendText(_ context.Context, r *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error) {
	m.textReqs = append(m.textReqs, r)

	return &sharedmodel.SendTextResponse{}, nil
}

func (m *fakeMessenger) SendImage(_ context.Context, r *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) {
	m.imageReqs = append(m.imageReqs, r)

	return &sharedmodel.SendImageResponse{}, nil
}

func (m *fakeMessenger) SendDocument(_ context.Context, r *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error) {
	m.documentReqs = append(m.documentReqs, r)

	return &sharedmodel.SendDocumentResponse{}, nil
}

func (m *fakeMessenger) SendLocation(_ context.Context, _ *sharedmodel.SendLocationRequest) (*sharedmodel.SendResponse, error) {
	return &sharedmodel.SendResponse{}, nil
}

func (m *fakeMessenger) SendContact(_ context.Context, _ *sharedmodel.SendContactRequest) (*sharedmodel.SendResponse, error) {
	return &sharedmodel.SendResponse{}, nil
}

func (m *fakeMessenger) SendInteractiveCallback(_ context.Context, _ *sharedmodel.SendInteractiveCallbackRequest) error {
	return nil
}

func (m *fakeMessenger) UpdateMessageDelivery(_ context.Context, _ *sharedmodel.MessageDeliveryReport) error {
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func gateWith(id, senderName, secret string) *vibbmmodel.ViberBMGate {
	return &vibbmmodel.ViberBMGate{
		ID:            id,
		SenderName:    senderName,
		WebhookSecret: secret,
		Enabled:       true,
		Peer:          sharedmodel.Peer{Sub: "bot-sub", Iss: "viber_bm"},
	}
}

// providerForReceipts builds a provider that can run processReceipt /
// processSeen / isSeenStatus without any external I/O.
func providerForReceipts(status *fakeStatusReporter) *viberBMProvider {
	return &viberBMProvider{
		logger: discardLogger(),
		status: status,
	}
}

// providerForMessages adds a messenger so processMessage can route text.
// userCache must be non-nil because syncContact calls IsKnown before any
// gateway round-trip; fakeUserCache always reports unknown so the call
// proceeds without a real Redis or gateway connection.
func providerForMessages(msgs *fakeMessenger) *viberBMProvider {
	return &viberBMProvider{
		logger:    discardLogger(),
		status:    &fakeStatusReporter{},
		messenger: msgs,
		userCache: fakeUserCache{},
	}
}

// ---------------------------------------------------------------------------
// parseInfobipTime
// ---------------------------------------------------------------------------

func TestParseInfobipTime(t *testing.T) {
	cases := []struct {
		name  string
		input string
		check func(t *testing.T, got time.Time)
	}{
		{
			name:  "infobip layout",
			input: "2024-05-01T12:00:00.000+0000",
			check: func(t *testing.T, got time.Time) {
				t.Helper()

				if got.Year() != 2024 || got.Month() != 5 || got.Day() != 1 {
					t.Errorf("unexpected date: %v", got)
				}
			},
		},
		{
			name:  "rfc3339 fallback",
			input: "2024-05-01T12:00:00Z",
			check: func(t *testing.T, got time.Time) {
				t.Helper()

				if got.Year() != 2024 {
					t.Errorf("unexpected year: %v", got)
				}
			},
		},
		{
			name:  "empty falls back to now",
			input: "",
			check: func(t *testing.T, got time.Time) {
				t.Helper()

				before := time.Now().Add(-time.Second)
				if got.Before(before) {
					t.Errorf("expected ~now, got %v", got)
				}
			},
		},
		{
			name:  "garbage falls back to now",
			input: "not-a-date",
			check: func(t *testing.T, got time.Time) {
				t.Helper()

				if got.IsZero() {
					t.Error("expected non-zero time for unparseable input")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, parseInfobipTime(tc.input))
		})
	}
}

// ---------------------------------------------------------------------------
// inboundResult.UnmarshalJSON — tolerant from/sender decode
// ---------------------------------------------------------------------------

func TestInboundResult_UnmarshalJSON_ToleratesSender(t *testing.T) {
	cases := []struct {
		name     string
		payload  string
		wantFrom string
	}{
		{
			name:     "from field",
			payload:  `{"from":"385991000001","to":"ViberBM","messageId":"m1"}`,
			wantFrom: "385991000001",
		},
		{
			name:     "sender alias",
			payload:  `{"sender":"385991000002","to":"ViberBM","messageId":"m2"}`,
			wantFrom: "385991000002",
		},
		{
			name:     "from wins over sender when both present",
			payload:  `{"from":"385991000001","sender":"385991000099","to":"ViberBM","messageId":"m3"}`,
			wantFrom: "385991000001",
		},
		{
			name:     "neither present yields empty from",
			payload:  `{"to":"ViberBM","messageId":"m4"}`,
			wantFrom: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r inboundResult
			if err := json.Unmarshal([]byte(tc.payload), &r); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if r.From != tc.wantFrom {
				t.Errorf("From = %q, want %q", r.From, tc.wantFrom)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// inboundBatch JSON — multiple results in one batch
// ---------------------------------------------------------------------------

func TestInboundBatch_MultipleResults(t *testing.T) {
	raw := `{
		"results": [
			{"from":"111","to":"SenderA","messageId":"m-1","message":{"type":"TEXT","text":"hi"}},
			{"from":"222","to":"SenderB","messageId":"m-2","status":{"groupId":5,"groupName":"DELIVERED","name":"DELIVERED_TO_HANDSET","description":"ok"},"doneAt":"2024-05-01T10:00:00.000+0000"},
			{"from":"333","to":"SenderC","messageId":"m-3","seenAt":"2024-05-01T11:00:00.000+0000"}
		]
	}`

	var batch inboundBatch
	if err := json.Unmarshal([]byte(raw), &batch); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(batch.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(batch.Results))
	}

	r0 := batch.Results[0]
	if r0.Message == nil || r0.Message.Text != "hi" {
		t.Errorf("result[0] message wrong: %+v", r0.Message)
	}

	r1 := batch.Results[1]
	if r1.Status == nil || r1.Status.GroupName != "DELIVERED" {
		t.Errorf("result[1] status wrong: %+v", r1.Status)
	}

	r2 := batch.Results[2]
	if r2.SeenAt == "" {
		t.Errorf("result[2] seenAt missing")
	}
}

// ---------------------------------------------------------------------------
// isSeenStatus
// ---------------------------------------------------------------------------

func TestIsSeenStatus(t *testing.T) {
	cases := []struct {
		name string
		s    *inboundStatus
		want bool
	}{
		{"nil", nil, false},
		{"SEEN uppercase", &inboundStatus{GroupName: "SEEN"}, true},
		{"seen lowercase", &inboundStatus{GroupName: "seen"}, true},
		{"Mixed case", &inboundStatus{GroupName: "Seen"}, true},
		{"DELIVERED is not seen", &inboundStatus{GroupName: "DELIVERED"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSeenStatus(tc.s); got != tc.want {
				t.Errorf("isSeenStatus(%v) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// processReceipt — DLR routing
// ---------------------------------------------------------------------------

func TestProcessReceipt(t *testing.T) {
	doneAt := "2024-05-01T12:00:00.000+0000"

	cases := []struct {
		name          string
		res           *inboundResult
		wantDelivered bool
		wantFailed    bool
		wantNone      bool
	}{
		{
			name: "DELIVERED triggers DeliveredByProviderIDs",
			res: &inboundResult{
				MessageID: "dlr-1",
				DoneAt:    doneAt,
				Status:    &inboundStatus{GroupName: "DELIVERED", Name: "DELIVERED_TO_HANDSET"},
			},
			wantDelivered: true,
		},
		{
			name: "REJECTED triggers FailedByProviderID",
			res: &inboundResult{
				MessageID: "dlr-2",
				DoneAt:    doneAt,
				Status:    &inboundStatus{GroupName: "REJECTED", Name: "REJECTED_OPERATOR", Description: "operator rejected"},
			},
			wantFailed: true,
		},
		{
			name: "EXPIRED triggers FailedByProviderID",
			res: &inboundResult{
				MessageID: "dlr-3",
				DoneAt:    doneAt,
				Status:    &inboundStatus{GroupName: "EXPIRED", Name: "EXPIRED", Description: "timed out"},
			},
			wantFailed: true,
		},
		{
			name: "UNDELIVERABLE triggers FailedByProviderID",
			res: &inboundResult{
				MessageID: "dlr-4",
				DoneAt:    doneAt,
				Status:    &inboundStatus{GroupName: "UNDELIVERABLE", Name: "UNREACHABLE", Description: "user unreachable"},
			},
			wantFailed: true,
		},
		{
			name: "lowercase delivered is case-insensitive",
			res: &inboundResult{
				MessageID: "dlr-ci",
				DoneAt:    doneAt,
				Status:    &inboundStatus{GroupName: "delivered"},
			},
			wantDelivered: true,
		},
		{
			name: "PENDING is ignored as transient",
			res: &inboundResult{
				MessageID: "dlr-5",
				DoneAt:    doneAt,
				Status:    &inboundStatus{GroupName: "PENDING", Name: "PENDING_ENROUTE"},
			},
			wantNone: true,
		},
		{
			name:     "empty messageID skips",
			res:      &inboundResult{MessageID: "", Status: &inboundStatus{GroupName: "DELIVERED"}},
			wantNone: true,
		},
		{
			name:     "nil status skips",
			res:      &inboundResult{MessageID: "dlr-6", Status: nil},
			wantNone: true,
		},
	}

	gate := gateWith("g-dlr", "ViberBM", "")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := &fakeStatusReporter{}
			p := providerForReceipts(status)

			p.processReceipt(context.Background(), gate, tc.res)

			switch {
			case tc.wantDelivered:
				if len(status.deliveredIDs) == 0 {
					t.Fatal("expected DeliveredByProviderIDs to be called")
				}

				if status.deliveredIDs[0] != tc.res.MessageID {
					t.Errorf("deliveredID = %q, want %q", status.deliveredIDs[0], tc.res.MessageID)
				}

				if status.failedID != "" {
					t.Errorf("FailedByProviderID must not be called for DELIVERED")
				}
			case tc.wantFailed:
				if status.failedID == "" {
					t.Fatal("expected FailedByProviderID to be called")
				}

				if status.failedID != tc.res.MessageID {
					t.Errorf("failedID = %q, want %q", status.failedID, tc.res.MessageID)
				}

				if status.failedCode != tc.res.Status.Name {
					t.Errorf("failedCode = %q, want %q", status.failedCode, tc.res.Status.Name)
				}

				if len(status.deliveredIDs) != 0 {
					t.Errorf("DeliveredByProviderIDs must not be called for %s", tc.res.Status.GroupName)
				}
			case tc.wantNone:
				if len(status.deliveredIDs) != 0 || status.failedID != "" {
					t.Errorf("expected no status calls; delivered=%v failed=%q", status.deliveredIDs, status.failedID)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// processSeen
// ---------------------------------------------------------------------------

func TestProcessSeen(t *testing.T) {
	cases := []struct {
		name      string
		res       *inboundResult
		wantRead  bool
		wantEmpty bool
	}{
		{
			name:     "seenAt populated",
			res:      &inboundResult{To: "385991000001", SeenAt: "2024-05-01T12:00:00.000+0000"},
			wantRead: true,
		},
		{
			name:     "status GroupName SEEN",
			res:      &inboundResult{To: "385991000002", Status: &inboundStatus{GroupName: "SEEN"}},
			wantRead: true,
		},
		{
			name:      "empty To skips even with seenAt",
			res:       &inboundResult{To: "", SeenAt: "2024-05-01T12:00:00.000+0000"},
			wantEmpty: true,
		},
	}

	gate := gateWith("g-seen", "ViberBM", "")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := &fakeStatusReporter{}
			p := providerForReceipts(status)

			p.processSeen(context.Background(), gate, tc.res)

			if tc.wantRead {
				if status.readUpTo == "" {
					t.Fatal("expected ReadUpTo to be called")
				}

				if status.readUpTo != tc.res.To {
					t.Errorf("readUpTo = %q, want %q", status.readUpTo, tc.res.To)
				}

				if status.readGate != gate.ID {
					t.Errorf("readGate = %q, want %q", status.readGate, gate.ID)
				}
			}

			if tc.wantEmpty && status.readUpTo != "" {
				t.Errorf("expected no ReadUpTo call, got %q", status.readUpTo)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// processMessage — bot (To) peer must never have Via set
// ---------------------------------------------------------------------------

func TestProcessMessage_BotPeerViaIsNil(t *testing.T) {
	msgs := &fakeMessenger{}
	p := providerForMessages(msgs)

	gate := gateWith("g1", "ViberBM", "")
	res := &inboundResult{
		From:      "385991000001",
		To:        "ViberBM",
		MessageID: "", // empty mid bypasses Redis (messageSeen returns false for "")
		Message:   &inboundMessage{Type: "TEXT", Text: "hello"},
	}

	if err := p.processMessage(context.Background(), gate, res); err != nil {
		t.Fatalf("processMessage: %v", err)
	}

	if len(msgs.textReqs) == 0 {
		t.Fatal("expected SendText to be called")
	}

	botPeer := msgs.textReqs[0].To
	if botPeer.Via != nil {
		t.Errorf("bot peer Via must be nil; got %v — this would misroute outbound replies", *botPeer.Via)
	}
}

// TestProcessMessage_EmptyFrom checks that a result with no From is dropped silently.
func TestProcessMessage_EmptyFromIsNoOp(t *testing.T) {
	msgs := &fakeMessenger{}
	p := providerForMessages(msgs)

	gate := gateWith("g1", "ViberBM", "")
	res := &inboundResult{
		From:      "",
		MessageID: "",
		Message:   &inboundMessage{Type: "TEXT", Text: "hello"},
	}

	if err := p.processMessage(context.Background(), gate, res); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs.textReqs) != 0 {
		t.Errorf("expected no SendText for empty From, got %d calls", len(msgs.textReqs))
	}
}

// ---------------------------------------------------------------------------
// processResult — dispatch priority
// ---------------------------------------------------------------------------

func TestProcessResult_Dispatch(t *testing.T) {
	gate := gateWith("g1", "ViberBM", "")

	t.Run("Message takes priority over Status", func(t *testing.T) {
		status := &fakeStatusReporter{}
		msgs := &fakeMessenger{}
		p := providerForMessages(msgs)
		p.status = status

		res := &inboundResult{
			From:      "111",
			MessageID: "",
			Message:   &inboundMessage{Type: "TEXT", Text: "hello"},
			Status:    &inboundStatus{GroupName: "DELIVERED"},
		}
		if err := p.processResult(context.Background(), gate, res); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(msgs.textReqs) == 0 {
			t.Error("expected SendText when Message is set")
		}

		if len(status.deliveredIDs) != 0 {
			t.Error("receipt must not fire when Message is set")
		}
	})

	t.Run("nil all fields is no-op", func(t *testing.T) {
		status := &fakeStatusReporter{}
		p := providerForReceipts(status)

		res := &inboundResult{From: "111", MessageID: "x"}
		if err := p.processResult(context.Background(), gate, res); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(status.deliveredIDs) != 0 || status.failedID != "" || status.readUpTo != "" {
			t.Error("expected no calls for result with no populated fields")
		}
	})

	t.Run("SeenAt routes to processSeen not processReceipt", func(t *testing.T) {
		status := &fakeStatusReporter{}
		p := providerForReceipts(status)

		res := &inboundResult{
			From:      "IBSelfServe",
			To:        "555",
			MessageID: "s-1",
			SeenAt:    "2024-05-01T12:00:00.000+0000",
			Status:    &inboundStatus{GroupName: "DELIVERED"},
		}
		if err := p.processResult(context.Background(), gate, res); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status.readUpTo == "" {
			t.Error("expected ReadUpTo (Seen path) to be called")
		}

		if len(status.deliveredIDs) != 0 {
			t.Error("Delivered must not fire when SeenAt is set")
		}
	})
}

// ---------------------------------------------------------------------------
// webhookURI normalisation (tested via webhookURI method directly)
// ---------------------------------------------------------------------------

func TestWebhookURI_Normalisation(t *testing.T) {
	p := &viberBMProvider{logger: discardLogger()}

	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"slash-prefixed is trimmed", "/viber/abc", "viber/abc"},
		{"no leading slash kept as-is", "viber/abc", "viber/abc"},
		{"empty stays empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), provider.WebhookURIKey, tc.value)
			if got := p.webhookURI(ctx); got != tc.want {
				t.Errorf("webhookURI = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// toExternalUser
// ---------------------------------------------------------------------------

func TestToExternalUser(t *testing.T) {
	t.Run("display name used when present", func(t *testing.T) {
		u := toExternalUser("385991000001", "Alice")
		if u.FirstName != "Alice" {
			t.Errorf("FirstName = %q, want Alice", u.FirstName)
		}

		if u.ID != "385991000001" {
			t.Errorf("ID = %q, want 385991000001", u.ID)
		}
	})

	t.Run("falls back to MSISDN when name empty", func(t *testing.T) {
		u := toExternalUser("385991000001", "")
		if u.FirstName != "385991000001" {
			t.Errorf("FirstName should fall back to MSISDN, got %q", u.FirstName)
		}
	})
}
