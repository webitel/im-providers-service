package viberbm

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pbstorage "github.com/webitel/im-providers-service/gen/go/storage"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

type fakeLinker struct {
	res   *pbstorage.GenerateFileLinkResponse
	err   error
	req   *pbstorage.GenerateFileLinkRequest
	calls int
}

func (f *fakeLinker) GenerateFileLink(_ context.Context, in *pbstorage.GenerateFileLinkRequest) (*pbstorage.GenerateFileLinkResponse, error) {
	f.req = in
	f.calls++

	return f.res, f.err
}

func videoLink(durationMs, size int64, withThumb bool) *pbstorage.GenerateFileLinkResponse {
	meta := &pbstorage.GenerateFileLinkResponse_Metadata{
		Id:         42,
		MimeType:   "video/mp4",
		Size:       size,
		Properties: &pbstorage.CustomFileProperties{Duration: durationMs},
	}
	if withThumb {
		meta.Thumbnail = &pbstorage.Thumbnail{MimeType: "image/png", Size: 10}
	}

	return &pbstorage.GenerateFileLinkResponse{
		BaseUrl:  "https://dev.example.com",
		Url:      "/api/storage/any/file/download?uuid=42&source=file&fetch_thumbnail=true&signature=abc",
		Metadata: meta,
	}
}

func TestClassifyMedia(t *testing.T) {
	cases := []struct {
		name     string
		mime     string
		fileName string
		want     mediaKind
	}{
		{"image mime", "image/jpeg", "a.jpg", mediaImage},
		{"video mime", "video/mp4", "a.mp4", mediaVideo},
		{"audio mime", "audio/mpeg", "a.mp3", mediaAudio},
		{"pdf mime", "application/pdf", "a.pdf", mediaFile},
		{"explicit mime beats image extension", "application/pdf", "pexels.jpg", mediaFile},
		{"explicit mime beats video extension", "image/png", "clip.mp4", mediaImage},
		{"mime params ignored", "video/mp4; codecs=avc1", "x", mediaVideo},
		{"empty mime falls back to extension", "", "clip.MOV", mediaVideo},
		{"octet-stream falls back to extension", "application/octet-stream", "voice.ogg", mediaAudio},
		{"unknown extension is a file", "", "report.pdf", mediaFile},
		{"nothing known is a file", "", "", mediaFile},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyMedia(tc.mime, tc.fileName); got != tc.want {
				t.Errorf("classifyMedia(%q, %q) = %v, want %v", tc.mime, tc.fileName, got, tc.want)
			}
		})
	}
}

func TestIsSupportedVideo(t *testing.T) {
	cases := []struct {
		name     string
		mime     string
		fileName string
		want     bool
	}{
		{"mp4", "video/mp4", "a.mp4", true},
		{"quicktime", "video/quicktime", "a.mov", true},
		{"3gpp", "video/3gpp", "a.3gp", true},
		{"webm rejected", "video/webm", "a.webm", false},
		{"explicit webm mime beats mp4 name", "video/webm", "a.mp4", false},
		{"no mime, m4v extension", "", "a.m4v", true},
		{"no mime, mkv extension rejected", "", "a.mkv", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSupportedVideo(tc.mime, tc.fileName); got != tc.want {
				t.Errorf("isSupportedVideo(%q, %q) = %v, want %v", tc.mime, tc.fileName, got, tc.want)
			}
		})
	}
}

func TestISODuration(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{10 * time.Second, "PT10S"},
		{10*time.Second + time.Millisecond, "PT11S"},
		{300 * time.Millisecond, "PT1S"},
		{600 * time.Second, "PT600S"},
	}

	for _, tc := range cases {
		if got := isoDuration(tc.in); got != tc.want {
			t.Errorf("isoDuration(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSendDocument_Video(t *testing.T) {
	const okBody = `{"bulkId":"b","messages":[{"messageId":"m","status":{"groupId":1,"groupName":"PENDING","name":"PENDING_ENROUTE","description":""}}]}`

	video := &sharedmodel.Document{ID: "42", URL: "https://example.com/clip.mp4", FileName: "clip.mp4", MimeType: "video/mp4"}

	cases := []struct {
		name       string
		doc        *sharedmodel.Document
		links      *fakeLinker
		wantErr    error
		wantAnyErr bool
		wantLinks  int
	}{
		{
			name:      "mp4 sent as VIDEO with storage preview",
			doc:       video,
			links:     &fakeLinker{res: videoLink(12500, 1<<20, true)},
			wantLinks: 1,
		},
		{
			name:    "audio rejected before storage",
			doc:     &sharedmodel.Document{ID: "42", URL: "https://example.com/v.mp3", FileName: "v.mp3", MimeType: "audio/mpeg"},
			links:   &fakeLinker{},
			wantErr: vibbmmodel.ErrAudioUnsupported,
		},
		{
			name:    "webm rejected before storage",
			doc:     &sharedmodel.Document{ID: "42", URL: "https://example.com/v.webm", FileName: "v.webm", MimeType: "video/webm"},
			links:   &fakeLinker{},
			wantErr: vibbmmodel.ErrFileTypeUnsupported,
		},
		{
			name:    "non-storage id has no preview",
			doc:     &sharedmodel.Document{URL: "https://example.com/clip.mp4", FileName: "clip.mp4", MimeType: "video/mp4"},
			links:   &fakeLinker{},
			wantErr: vibbmmodel.ErrVideoPreviewUnavailable,
		},
		{
			name:      "missing thumbnail",
			doc:       video,
			links:     &fakeLinker{res: videoLink(12500, 1<<20, false)},
			wantErr:   vibbmmodel.ErrVideoPreviewUnavailable,
			wantLinks: 1,
		},
		{
			name: "empty thumbnail",
			doc:  video,
			links: &fakeLinker{res: func() *pbstorage.GenerateFileLinkResponse {
				r := videoLink(12500, 1<<20, true)
				r.Metadata.Thumbnail.Size = 0

				return r
			}()},
			wantErr:   vibbmmodel.ErrVideoPreviewUnavailable,
			wantLinks: 1,
		},
		{
			name:    "non-numeric id has no preview",
			doc:     &sharedmodel.Document{ID: "abc", URL: "https://example.com/clip.mp4", FileName: "clip.mp4", MimeType: "video/mp4"},
			links:   &fakeLinker{},
			wantErr: vibbmmodel.ErrVideoPreviewUnavailable,
		},
		{
			name:      "missing duration",
			doc:       video,
			links:     &fakeLinker{res: videoLink(0, 1<<20, true)},
			wantErr:   vibbmmodel.ErrVideoPreviewUnavailable,
			wantLinks: 1,
		},
		{
			name:      "longer than 600s rejected",
			doc:       video,
			links:     &fakeLinker{res: videoLink(601_000, 1<<20, true)},
			wantErr:   vibbmmodel.ErrVideoTooLong,
			wantLinks: 1,
		},
		{
			name:      "larger than 200MB rejected",
			doc:       video,
			links:     &fakeLinker{res: videoLink(12500, maxVideoBytes+1, true)},
			wantErr:   vibbmmodel.ErrVideoTooLarge,
			wantLinks: 1,
		},
		{
			name: "relative storage link without base fails",
			doc:  video,
			links: &fakeLinker{res: func() *pbstorage.GenerateFileLinkResponse {
				r := videoLink(1000, 1<<20, true)
				r.BaseUrl = ""

				return r
			}()},
			wantAnyErr: true,
			wantLinks:  1,
		},
		{
			name:       "link generation error fails",
			doc:        video,
			links:      &fakeLinker{err: errors.New("storage down")},
			wantAnyErr: true,
			wantLinks:  1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, cap := serveJSON(t, 200, okBody)

			p := &viberBMProvider{
				api:    apiClientForServer(srv),
				logger: discardLogger(),
				repo:   &signatureRepo{gate: &vibbmmodel.ViberBMGate{ID: "g1", DomainID: 7, BaseURL: srv.URL, APIKey: "key", SenderName: "Sender"}},
				links:  tc.links,
			}

			_, err := p.SendDocument(context.Background(), &sharedmodel.Message{
				GateID:    "g1",
				To:        sharedmodel.Peer{Sub: "38599000001"},
				Text:      "caption",
				Documents: []*sharedmodel.Document{tc.doc},
			})

			if tc.links.calls != tc.wantLinks {
				t.Errorf("link calls = %d, want %d", tc.links.calls, tc.wantLinks)
			}

			if tc.wantErr != nil || tc.wantAnyErr {
				if err == nil || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
					t.Fatalf("SendDocument err = %v, want %v", err, tc.wantErr)
				}

				if cap.body != nil {
					t.Error("request reached Infobip despite error")
				}

				return
			}

			if err != nil {
				t.Fatalf("SendDocument: %v", err)
			}

			req := tc.links.req
			if req.GetFileId() != 42 || req.GetDomainId() != 7 || req.GetSource() != "file" || req.GetAction() != "download" || !req.GetMetadata() || req.GetQuery()["fetch_thumbnail"] != "true" {
				t.Errorf("link request = %+v", req)
			}

			var env envelope
			if err := json.Unmarshal(cap.body, &env); err != nil {
				t.Fatalf("parse body: %v", err)
			}

			content, _ := env.Messages[0].Content.(map[string]any)
			want := map[string]any{
				"type":          contentTypeVideo,
				"mediaUrl":      video.URL,
				"mediaDuration": "PT13S",
				"thumbnailUrl":  "https://dev.example.com/api/storage/any/file/download?uuid=42&source=file&fetch_thumbnail=true&signature=abc",
				"text":          "caption",
			}

			for k, v := range want {
				if content[k] != v {
					t.Errorf("content.%s = %v, want %v", k, content[k], v)
				}
			}
		})
	}
}
