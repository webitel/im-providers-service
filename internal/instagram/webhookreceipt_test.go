package instagram

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

type fakeStatusReporter struct {
	deliveredIDs  []string
	deliveredAt   time.Time
	deliveredUpTo string
	upToWatermark time.Time
	readUpTo      string
	readWatermark time.Time
}

func (f *fakeStatusReporter) DeliveredByProviderIDs(_ context.Context, _ string, providerMessageIDs []string, at time.Time) {
	f.deliveredIDs = providerMessageIDs
	f.deliveredAt = at
}

func (f *fakeStatusReporter) DeliveredUpTo(_ context.Context, _, providerUserID string, watermark time.Time) {
	f.deliveredUpTo = providerUserID
	f.upToWatermark = watermark
}

func (f *fakeStatusReporter) ReadUpTo(_ context.Context, _, providerUserID string, watermark time.Time) {
	f.readUpTo = providerUserID
	f.readWatermark = watermark
}

func receiptProvider(status statusReporter) *instagramProvider {
	return &instagramProvider{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		status: status,
	}
}

func TestWatermarkTime(t *testing.T) {
	cases := []struct {
		name      string
		watermark int64
		timestamp int64
		want      time.Time
	}{
		{"watermark wins", 1_770_000_000_000, 1_760_000_000_000, time.UnixMilli(1_770_000_000_000)},
		{"falls back to event timestamp", 0, 1_760_000_000_000, time.UnixMilli(1_760_000_000_000)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := watermarkTime(tc.watermark, tc.timestamp); !got.Equal(tc.want) {
				t.Errorf("watermarkTime(%d, %d) = %v, want %v", tc.watermark, tc.timestamp, got, tc.want)
			}
		})
	}

	t.Run("falls back to now", func(t *testing.T) {
		before := time.Now()
		got := watermarkTime(0, 0)

		if got.Before(before) || got.After(time.Now()) {
			t.Errorf("expected ~now, got %v", got)
		}
	})
}

func TestMessageID(t *testing.T) {
	cases := []struct {
		name string
		msg  Messaging
		want string
	}{
		{"message mid", Messaging{Message: &InboundMessage{Mid: "m-1"}}, "m-1"},
		{"postback mid", Messaging{Postback: &Postback{Mid: "p-1"}}, "p-1"},
		{"receipt-only event", Messaging{Delivery: &Delivery{}}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := messageID(tc.msg); got != tc.want {
				t.Errorf("messageID() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProcessReceipt_DeliveryWithMids(t *testing.T) {
	status := &fakeStatusReporter{}
	p := receiptProvider(status)

	p.processReceipt(context.Background(), &igmodel.InstagramGate{ID: "gate-1"}, "igsid-1", Messaging{
		Timestamp: 1_760_000_000_000,
		Delivery:  &Delivery{Mids: []string{"mid.a", "mid.b"}, Watermark: 1_770_000_000_000},
	})

	if len(status.deliveredIDs) != 2 || status.deliveredIDs[0] != "mid.a" {
		t.Fatalf("expected mids to be reported directly, got %v", status.deliveredIDs)
	}

	if !status.deliveredAt.Equal(time.UnixMilli(1_770_000_000_000)) {
		t.Errorf("expected watermark time, got %v", status.deliveredAt)
	}

	if status.deliveredUpTo != "" {
		t.Errorf("watermark path must not run when mids are present, got %q", status.deliveredUpTo)
	}
}

func TestProcessReceipt_DeliveryWatermarkOnly(t *testing.T) {
	status := &fakeStatusReporter{}
	p := receiptProvider(status)

	p.processReceipt(context.Background(), &igmodel.InstagramGate{ID: "gate-1"}, "igsid-1", Messaging{
		Delivery: &Delivery{Watermark: 1_770_000_000_000},
	})

	if status.deliveredUpTo != "igsid-1" {
		t.Fatalf("expected watermark delivery for igsid-1, got %q", status.deliveredUpTo)
	}

	if len(status.deliveredIDs) != 0 {
		t.Errorf("mids path must not run without mids, got %v", status.deliveredIDs)
	}
}

func TestProcessReceipt_ReadWatermark(t *testing.T) {
	status := &fakeStatusReporter{}
	p := receiptProvider(status)

	p.processReceipt(context.Background(), &igmodel.InstagramGate{ID: "gate-1"}, "igsid-9", Messaging{
		Read: &Read{Watermark: 1_770_000_222_000},
	})

	if status.readUpTo != "igsid-9" {
		t.Fatalf("expected read watermark for igsid-9, got %q", status.readUpTo)
	}

	if !status.readWatermark.Equal(time.UnixMilli(1_770_000_222_000)) {
		t.Errorf("expected watermark time, got %v", status.readWatermark)
	}
}

func TestProcessReceipt_DeliveryAndReadTogether(t *testing.T) {
	status := &fakeStatusReporter{}
	p := receiptProvider(status)

	p.processReceipt(context.Background(), &igmodel.InstagramGate{ID: "gate-1"}, "igsid-1", Messaging{
		Delivery: &Delivery{Watermark: 1},
		Read:     &Read{Watermark: 2},
	})

	if status.deliveredUpTo == "" || status.readUpTo == "" {
		t.Fatalf("both receipts must be routed: delivered=%q read=%q", status.deliveredUpTo, status.readUpTo)
	}
}
