package viberbm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pbstorage "github.com/webitel/im-providers-service/gen/go/storage"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

type fakeProber struct {
	meta  *videoMeta
	err   error
	calls int
}

func (f *fakeProber) Probe(_ context.Context, _ string) (*videoMeta, error) {
	f.calls++

	return f.meta, f.err
}

type fakeMedia struct {
	url  string
	err  error
	req  sharedmodel.UploadRequest
	body []byte
}

func (f *fakeMedia) UploadFile(_ context.Context, req sharedmodel.UploadRequest, body io.Reader) (sharedmodel.UploadResponse, error) {
	f.req = req
	f.body, _ = io.ReadAll(body)

	return sharedmodel.UploadResponse{ID: "1", URL: f.url}, f.err
}

type fakeLinker struct {
	baseURL, url string
	err          error
	req          *pbstorage.GenerateFileLinkRequest
}

func (f *fakeLinker) GenerateFileLink(_ context.Context, in *pbstorage.GenerateFileLinkRequest) (*pbstorage.GenerateFileLinkResponse, error) {
	f.req = in

	return &pbstorage.GenerateFileLinkResponse{BaseUrl: f.baseURL, Url: f.url}, f.err
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

	video := &sharedmodel.Document{URL: "https://example.com/clip.mp4", FileName: "clip.mp4", MimeType: "video/mp4"}

	cases := []struct {
		name       string
		doc        *sharedmodel.Document
		prober     *fakeProber
		media      *fakeMedia
		links      *fakeLinker
		wantErr    error
		wantAnyErr bool
		wantProbes int
	}{
		{
			name:       "mp4 sent as VIDEO",
			doc:        video,
			prober:     &fakeProber{meta: &videoMeta{duration: 12500 * time.Millisecond, thumbnail: []byte("jpeg")}},
			media:      &fakeMedia{url: "/relative/thumb.jpg"},
			links:      &fakeLinker{baseURL: "https://dev.example.com", url: "/api/storage/file/42/download?signature=abc"},
			wantProbes: 1,
		},
		{
			name:    "audio rejected before Infobip",
			doc:     &sharedmodel.Document{URL: "https://example.com/v.mp3", FileName: "v.mp3", MimeType: "audio/mpeg"},
			prober:  &fakeProber{},
			media:   &fakeMedia{},
			wantErr: vibbmmodel.ErrAudioUnsupported,
		},
		{
			name:    "webm rejected before probing",
			doc:     &sharedmodel.Document{URL: "https://example.com/v.webm", FileName: "v.webm", MimeType: "video/webm"},
			prober:  &fakeProber{},
			media:   &fakeMedia{},
			wantErr: vibbmmodel.ErrFileTypeUnsupported,
		},
		{
			name:       "longer than 600s rejected",
			doc:        video,
			prober:     &fakeProber{meta: &videoMeta{duration: 601 * time.Second, thumbnail: []byte("jpeg")}},
			media:      &fakeMedia{url: "https://storage.example.com/thumb.jpg"},
			wantErr:    vibbmmodel.ErrVideoTooLong,
			wantProbes: 1,
		},
		{
			name:       "missing ffmpeg surfaces probe error",
			doc:        video,
			prober:     &fakeProber{err: vibbmmodel.ErrVideoProbeUnavailable},
			media:      &fakeMedia{},
			wantErr:    vibbmmodel.ErrVideoProbeUnavailable,
			wantProbes: 1,
		},
		{
			name:       "relative storage link without base fails",
			doc:        video,
			prober:     &fakeProber{meta: &videoMeta{duration: time.Second, thumbnail: []byte("jpeg")}},
			media:      &fakeMedia{},
			links:      &fakeLinker{url: "/api/storage/file/42/download"},
			wantAnyErr: true,
			wantProbes: 1,
		},
		{
			name:       "link generation error fails",
			doc:        video,
			prober:     &fakeProber{meta: &videoMeta{duration: time.Second, thumbnail: []byte("jpeg")}},
			media:      &fakeMedia{},
			links:      &fakeLinker{err: errors.New("storage down")},
			wantAnyErr: true,
			wantProbes: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, cap := serveJSON(t, 200, okBody)

			p := &viberBMProvider{
				api:    apiClientForServer(srv),
				logger: discardLogger(),
				repo:   &signatureRepo{gate: &vibbmmodel.ViberBMGate{ID: "g1", DomainID: 7, BaseURL: srv.URL, APIKey: "key", SenderName: "Sender"}},
				video:  tc.prober,
				media:  tc.media,
				links:  tc.links,
			}

			_, err := p.SendDocument(context.Background(), &sharedmodel.Message{
				GateID:    "g1",
				To:        sharedmodel.Peer{Sub: "38599000001"},
				Text:      "caption",
				Documents: []*sharedmodel.Document{tc.doc},
			})

			if tc.prober.calls != tc.wantProbes {
				t.Errorf("probe calls = %d, want %d", tc.prober.calls, tc.wantProbes)
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

			if tc.links.req.GetFileId() != 1 || tc.links.req.GetDomainId() != 7 || tc.links.req.GetAction() != "download" {
				t.Errorf("link request = %+v", tc.links.req)
			}

			if tc.media.req.DomainID != 7 || tc.media.req.MimeType != "image/jpeg" || !bytes.Equal(tc.media.body, []byte("jpeg")) {
				t.Errorf("thumbnail upload = %+v body %q", tc.media.req, tc.media.body)
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
				"thumbnailUrl":  "https://dev.example.com/api/storage/file/42/download?signature=abc",
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

// TestFFmpegProber_Probe runs the real binaries against a generated clip.
func TestFFmpegProber_Probe(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}

	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}

	clip := filepath.Join(t.TempDir(), "clip.mp4")

	gen := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=3:size=320x240:rate=10",
		"-pix_fmt", "yuv420p", clip)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate clip: %v: %s", err, out)
	}

	data, err := os.ReadFile(clip)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mirrors CDNs that block Go's default User-Agent.
		if strings.HasPrefix(r.UserAgent(), "Go-http-client") {
			w.WriteHeader(http.StatusForbidden)

			return
		}

		w.Header().Set("Content-Type", "video/mp4")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	prober := newFFmpegProber()
	prober.client = srv.Client()

	meta, err := prober.Probe(context.Background(), srv.URL+"/clip.mp4")
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	if meta.duration < 2900*time.Millisecond || meta.duration > 3100*time.Millisecond {
		t.Errorf("duration = %v, want ~3s", meta.duration)
	}

	if !bytes.HasPrefix(meta.thumbnail, []byte{0xFF, 0xD8}) {
		t.Errorf("thumbnail is not a JPEG (first bytes %x)", meta.thumbnail[:min(4, len(meta.thumbnail))])
	}
}

func TestFFmpegProber_MissingBinary(t *testing.T) {
	prober := newFFmpegProber()
	prober.ffprobe = "definitely-not-ffprobe"

	if _, err := prober.Probe(context.Background(), "https://example.com/a.mp4"); !errors.Is(err, vibbmmodel.ErrVideoProbeUnavailable) {
		t.Fatalf("Probe err = %v, want ErrVideoProbeUnavailable", err)
	}
}

func TestFFmpegProber_RejectsNonHTTPS(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}

	if _, err := newFFmpegProber().Probe(context.Background(), "file:///etc/passwd"); err == nil {
		t.Fatal("expected non-https url to be refused")
	}
}
