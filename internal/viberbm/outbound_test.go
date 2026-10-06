package viberbm

import (
	"context"
	"encoding/json"
	"errors"
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
// documentName — Infobip fileName rules
// ---------------------------------------------------------------------------

func TestDocumentName(t *testing.T) {
	cases := []struct {
		name    string
		doc     *sharedmodel.Document
		want    string
		wantErr error
	}{
		{"short name unchanged", &sharedmodel.Document{FileName: "short.pdf"}, "short.pdf", nil},
		{"extension lowercased", &sharedmodel.Document{FileName: "Report.PDF"}, "Report.pdf", nil},
		{"exactly 25 runes unchanged", &sharedmodel.Document{FileName: "123456789012345678901.pdf"}, "123456789012345678901.pdf", nil},
		{"long stem truncated, extension kept", &sharedmodel.Document{FileName: "quarterly-report-2026-final.pdf"}, "quarterly-report-2026.pdf", nil},
		{"cyrillic stem truncated by runes", &sharedmodel.Document{FileName: "звітзвітзвітзвітзвітзвітзвіт.docx"}, "звітзвітзвітзвітзвіт.docx", nil},
		{"missing extension taken from mime", &sharedmodel.Document{FileName: "report", MimeType: "application/pdf"}, "report.pdf", nil},
		{"unsupported extension replaced from mime", &sharedmodel.Document{FileName: "report.bin", MimeType: "application/pdf"}, "report.bin.pdf", nil},
		{"empty name falls back to file stem", &sharedmodel.Document{MimeType: "application/pdf"}, "file.pdf", nil},
		{"unsupported type rejected", &sharedmodel.Document{FileName: "archive.zip", MimeType: "application/zip"}, "", vibbmmodel.ErrFileTypeUnsupported},
		{"no extension and no mime rejected", &sharedmodel.Document{FileName: "report"}, "", vibbmmodel.ErrFileTypeUnsupported},
		{"nil document rejected", nil, "", vibbmmodel.ErrFileTypeUnsupported},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := documentName(tc.doc)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("documentName err = %v, want %v", err, tc.wantErr)
			}

			if got != tc.want {
				t.Errorf("documentName = %q, want %q", got, tc.want)
			}

			if len([]rune(got)) > maxFileNameLen {
				t.Errorf("documentName = %q exceeds %d runes", got, maxFileNameLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SendDocument — content type routing
// ---------------------------------------------------------------------------

func TestSendDocument_Routing(t *testing.T) {
	const okBody = `{"bulkId":"b","messages":[{"messageId":"m","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`

	cases := []struct {
		name         string
		doc          *sharedmodel.Document
		wantType     string
		wantFileName string
		wantText     string
		wantErr      error
	}{
		{
			name:     "image mime sent as IMAGE with caption",
			doc:      &sharedmodel.Document{URL: "https://example.com/p.jpg", FileName: "pexels.jpg", MimeType: "image/jpeg"},
			wantType: contentTypeImage,
			wantText: "caption",
		},
		{
			name:     "image detected by extension without mime",
			doc:      &sharedmodel.Document{URL: "https://example.com/p.png", FileName: "photo.png"},
			wantType: contentTypeImage,
			wantText: "caption",
		},
		{
			name:         "pdf sent as FILE",
			doc:          &sharedmodel.Document{URL: "https://example.com/r.pdf", FileName: "report.pdf", MimeType: "application/pdf"},
			wantType:     contentTypeFile,
			wantFileName: "report.pdf",
		},
		{
			name:    "unsupported file rejected before Infobip",
			doc:     &sharedmodel.Document{URL: "https://example.com/a.zip", FileName: "archive.zip", MimeType: "application/zip"},
			wantErr: vibbmmodel.ErrFileTypeUnsupported,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, cap := serveJSON(t, 200, okBody)

			p := &viberBMProvider{
				api:    apiClientForServer(srv),
				logger: discardLogger(),
				repo:   &signatureRepo{gate: &vibbmmodel.ViberBMGate{ID: "g1", DomainID: 1, BaseURL: srv.URL, APIKey: "key", SenderName: "Sender"}},
			}

			_, err := p.SendDocument(context.Background(), &sharedmodel.Message{
				GateID:    "g1",
				To:        sharedmodel.Peer{Sub: "38599000001"},
				Text:      "caption",
				Documents: []*sharedmodel.Document{tc.doc},
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("SendDocument err = %v, want %v", err, tc.wantErr)
			}

			if tc.wantErr != nil {
				if cap.body != nil {
					t.Error("request reached Infobip despite validation error")
				}

				return
			}

			var env envelope
			if err := json.Unmarshal(cap.body, &env); err != nil {
				t.Fatalf("parse body: %v", err)
			}

			content, _ := env.Messages[0].Content.(map[string]any)

			if got := content["type"]; got != tc.wantType {
				t.Errorf("content.type = %v, want %s", got, tc.wantType)
			}

			if got := content["mediaUrl"]; got != tc.doc.URL {
				t.Errorf("mediaUrl = %v, want %s", got, tc.doc.URL)
			}

			if tc.wantFileName != "" && content["fileName"] != tc.wantFileName {
				t.Errorf("fileName = %v, want %s", content["fileName"], tc.wantFileName)
			}

			if tc.wantText != "" && content["text"] != tc.wantText {
				t.Errorf("text = %v, want %s", content["text"], tc.wantText)
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
