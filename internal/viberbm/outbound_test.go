package viberbm

import (
	"context"
	"testing"

	"github.com/google/uuid"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// ---------------------------------------------------------------------------
// normalizeMSISDN
// ---------------------------------------------------------------------------

func TestNormalizeMSISDN(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"bare number passthrough", "38599000001", "38599000001"},
		{"leading plus stripped", "+38599000001", "38599000001"},
		{"surrounding whitespace trimmed", " 38599000001 ", "38599000001"},
		{"plus and spaces", " +38599000001 ", "38599000001"},
		{"empty stays empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeMSISDN(tc.input)
			if got != tc.want {
				t.Errorf("normalizeMSISDN(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// resolveMSISDN — non-UUID path (no network required)
// ---------------------------------------------------------------------------

func TestResolveMSISDN_NonUUID(t *testing.T) {
	cases := []struct {
		name string
		sub  string
		want string
	}{
		{"bare MSISDN passthrough", "38599000001", "38599000001"},
		{"plus stripped", "+38599000001", "38599000001"},
		{"spaces trimmed", " 38599000001 ", "38599000001"},
	}

	p := &viberBMProvider{logger: discardLogger()}
	gate := &vibbmmodel.ViberBMGate{ID: "g1", DomainID: 1}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := p.resolveMSISDN(context.Background(), gate, tc.sub)
			if err != nil {
				t.Fatalf("resolveMSISDN: unexpected error: %v", err)
			}

			if got != tc.want {
				t.Errorf("resolveMSISDN(%q) = %q, want %q", tc.sub, got, tc.want)
			}
		})
	}
}

// TestResolveMSISDN_UUID verifies that a hyphen-containing sub (UUID-shaped)
// takes the contact-search path and returns an error when contactClient is nil
// (i.e. a gRPC call would be needed — we don't wire the real client in unit tests).
func TestResolveMSISDN_UUID_RequiresContactClient(t *testing.T) {
	p := &viberBMProvider{
		logger:        discardLogger(),
		contactClient: nil, // intentionally nil — will panic/error on call
	}

	gate := &vibbmmodel.ViberBMGate{ID: "g1", DomainID: 1}
	uuidSub := uuid.New().String() // contains hyphens → UUID branch

	// Should return an error (nil contactClient dereference → panic recovered
	// by the defer/recover in the test, or nil-pointer at runtime). We wrap the
	// call to catch either outcome and assert it is not a clean success.
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				err = context.DeadlineExceeded // proxy sentinel: panic == error
			}
		}()

		_, err = p.resolveMSISDN(context.Background(), gate, uuidSub)
	}()

	if err == nil {
		t.Fatal("expected error/panic when contactClient is nil and sub is UUID-shaped; got nil error")
	}
}

// ---------------------------------------------------------------------------
// toResponse — ID/MD stamping
// ---------------------------------------------------------------------------

func TestToResponse(t *testing.T) {
	t.Run("nil result returns nil", func(t *testing.T) {
		resp, err := toResponse(nil, "38599", nil)
		if err != nil || resp != nil {
			t.Errorf("toResponse(nil, _, nil) = (%v, %v), want (nil, nil)", resp, err)
		}
	})

	t.Run("non-nil error propagates", func(t *testing.T) {
		sentErr := vibbmmodel.ErrMediaMissing
		resp, err := toResponse(nil, "38599", sentErr)

		if err != sentErr {
			t.Errorf("err = %v, want %v", err, sentErr)
		}

		if resp != nil {
			t.Errorf("resp should be nil on error, got %v", resp)
		}
	})

	t.Run("MessageID stamped into response ID", func(t *testing.T) {
		res := &sendResult{MessageID: "provider-msg-id", BulkID: ""}

		resp, err := toResponse(res, "38599", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.ID != "provider-msg-id" {
			t.Errorf("ID = %q, want provider-msg-id", resp.ID)
		}
	})

	t.Run("recipient_id always in MD", func(t *testing.T) {
		res := &sendResult{MessageID: "m", BulkID: ""}
		resp, _ := toResponse(res, "38599000001", nil)

		got, ok := resp.MD["recipient_id"]
		if !ok {
			t.Fatal("recipient_id missing from MD")
		}

		if got != "38599000001" {
			t.Errorf("recipient_id = %v, want 38599000001", got)
		}
	})

	t.Run("bulk_id in MD only when non-empty", func(t *testing.T) {
		resWith := &sendResult{MessageID: "m", BulkID: "bulk-abc"}
		respWith, _ := toResponse(resWith, "38599", nil)

		if got := respWith.MD["bulk_id"]; got != "bulk-abc" {
			t.Errorf("bulk_id = %v, want bulk-abc", got)
		}

		resWithout := &sendResult{MessageID: "m", BulkID: ""}
		respWithout, _ := toResponse(resWithout, "38599", nil)

		if _, ok := respWithout.MD["bulk_id"]; ok {
			t.Error("bulk_id should be absent when BulkID is empty")
		}
	})
}

// ---------------------------------------------------------------------------
// documentName — truncation and fallback
// ---------------------------------------------------------------------------

func TestDocumentName(t *testing.T) {
	cases := []struct {
		name string
		req  *sharedmodel.Message
		want string
	}{
		{
			name: "short name unchanged",
			req:  &sharedmodel.Message{Documents: []*sharedmodel.Document{{FileName: "short.pdf"}}},
			want: "short.pdf",
		},
		{
			name: "exactly 25 runes unchanged",
			req:  &sharedmodel.Message{Documents: []*sharedmodel.Document{{FileName: "1234567890123456789012345"}}},
			want: "1234567890123456789012345",
		},
		{
			name: "26 runes truncated to 25",
			req:  &sharedmodel.Message{Documents: []*sharedmodel.Document{{FileName: "12345678901234567890123456"}}},
			want: "1234567890123456789012345",
		},
		{
			name: "empty name falls back to 'file'",
			req:  &sharedmodel.Message{Documents: []*sharedmodel.Document{{FileName: ""}}},
			want: "file",
		},
		{
			name: "nil document falls back to 'file'",
			req:  &sharedmodel.Message{Documents: []*sharedmodel.Document{nil}},
			want: "file",
		},
		{
			name: "empty documents slice falls back to 'file'",
			req:  &sharedmodel.Message{Documents: nil},
			want: "file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := documentName(tc.req)
			if got != tc.want {
				t.Errorf("documentName = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// firstURL
// ---------------------------------------------------------------------------

func TestFirstURL(t *testing.T) {
	t.Run("empty slice returns empty string", func(t *testing.T) {
		got := firstURL([]*sharedmodel.Image{})
		if got != "" {
			t.Errorf("firstURL([]) = %q, want empty", got)
		}
	})

	t.Run("nil slice returns empty string", func(t *testing.T) {
		got := firstURL([]*sharedmodel.Image(nil))
		if got != "" {
			t.Errorf("firstURL(nil) = %q, want empty", got)
		}
	})

	t.Run("returns URL of first element", func(t *testing.T) {
		imgs := []*sharedmodel.Image{
			{URL: "https://example.com/first.jpg"},
			{URL: "https://example.com/second.jpg"},
		}

		got := firstURL(imgs)
		if got != "https://example.com/first.jpg" {
			t.Errorf("firstURL = %q, want first URL", got)
		}
	})

	t.Run("works with documents too", func(t *testing.T) {
		docs := []*sharedmodel.Document{{URL: "https://example.com/doc.pdf"}}

		got := firstURL(docs)
		if got != "https://example.com/doc.pdf" {
			t.Errorf("firstURL(docs) = %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// isImageMedia
// ---------------------------------------------------------------------------

func TestIsImageMedia(t *testing.T) {
	cases := []struct {
		name     string
		mime     string
		fileName string
		want     bool
	}{
		{"image/jpeg mime", "image/jpeg", "photo.jpg", true},
		{"image/png mime", "image/png", "img.png", true},
		{"image/webp mime", "image/webp", "photo.webp", true},
		{"application/pdf is not image", "application/pdf", "doc.pdf", false},
		{"empty mime — jpg extension", "", "photo.jpg", true},
		{"empty mime — jpeg extension", "", "photo.jpeg", true},
		{"empty mime — png extension", "", "photo.png", true},
		{"empty mime — gif extension", "", "anim.gif", true},
		{"empty mime — webp extension", "", "photo.webp", true},
		{"empty mime — pdf extension not image", "", "report.pdf", false},
		{"empty mime and empty name", "", "", false},
		{"uppercase extension matched case-insensitively", "", "PHOTO.JPG", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isImageMedia(tc.mime, tc.fileName)
			if got != tc.want {
				t.Errorf("isImageMedia(%q, %q) = %v, want %v", tc.mime, tc.fileName, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// dedup — messageSeen with nil Redis client (empty mid fast-path)
// ---------------------------------------------------------------------------

// TestMessageSeen_EmptyMID verifies the contract: empty messageId is never
// treated as a duplicate regardless of Redis state (no network call made).
func TestMessageSeen_EmptyMID(t *testing.T) {
	// rdb is intentionally nil; the function must return false before any
	// Redis call when mid == "".
	got := messageSeen(context.Background(), nil, "")
	if got {
		t.Error("messageSeen with empty mid must return false (not a duplicate)")
	}
}
