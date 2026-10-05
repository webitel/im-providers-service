package viberbm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// Infobip VIDEO content limits.
const (
	maxVideoBytes    = 200 << 20
	maxVideoDuration = 600 * time.Second
	thumbnailWidth   = 800
)

// mediaUserAgent replaces Go's default "Go-http-client/1.1", which common CDNs
// reject with 403 as a bot signature.
const mediaUserAgent = "webitel-im-providers"

type videoMeta struct {
	duration  time.Duration
	thumbnail []byte // JPEG
}

// videoProber derives the mediaDuration and thumbnail Infobip requires for
// VIDEO content but the outbound message does not carry.
type videoProber interface {
	Probe(ctx context.Context, videoURL string) (*videoMeta, error)
}

type ffmpegProber struct {
	client  *http.Client
	ffprobe string
	ffmpeg  string
}

// newFFmpegProber resolves the binaries lazily per call so a host without
// ffmpeg still serves every other content type.
func newFFmpegProber() *ffmpegProber {
	return &ffmpegProber{
		client:  newGuardedClient(2 * time.Minute),
		ffprobe: "ffprobe",
		ffmpeg:  "ffmpeg",
	}
}

func (f *ffmpegProber) Probe(ctx context.Context, videoURL string) (*videoMeta, error) {
	ffprobe, err := exec.LookPath(f.ffprobe)
	if err != nil {
		return nil, vibbmmodel.ErrVideoProbeUnavailable
	}

	ffmpeg, err := exec.LookPath(f.ffmpeg)
	if err != nil {
		return nil, vibbmmodel.ErrVideoProbeUnavailable
	}

	// Download through the guarded client rather than handing the URL to
	// ffmpeg, whose protocol handlers would bypass the SSRF dial guard.
	path, err := f.download(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)

	duration, err := probeDuration(ctx, ffprobe, path)
	if err != nil {
		return nil, err
	}

	thumbnail, err := extractThumbnail(ctx, ffmpeg, path, duration)
	if err != nil {
		return nil, err
	}

	return &videoMeta{duration: duration, thumbnail: thumbnail}, nil
}

func (f *ffmpegProber) download(ctx context.Context, videoURL string) (string, error) {
	if err := validateFetchURL(videoURL); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, videoURL, nil)
	if err != nil {
		return "", fmt.Errorf("viber_bm video download: %w", err)
	}

	req.Header.Set("User-Agent", mediaUserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("viber_bm video download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("viber_bm video download: status %s", resp.Status)
	}

	if resp.ContentLength > maxVideoBytes {
		return "", vibbmmodel.ErrVideoTooLarge
	}

	tmp, err := os.CreateTemp("", "viberbm-video-*")
	if err != nil {
		return "", fmt.Errorf("viber_bm video download: %w", err)
	}

	n, copyErr := io.Copy(tmp, io.LimitReader(resp.Body, maxVideoBytes+1))
	closeErr := tmp.Close()

	switch {
	case copyErr != nil:
		err = fmt.Errorf("viber_bm video download: %w", copyErr)
	case closeErr != nil:
		err = fmt.Errorf("viber_bm video download: %w", closeErr)
	case n > maxVideoBytes:
		err = vibbmmodel.ErrVideoTooLarge
	}

	if err != nil {
		_ = os.Remove(tmp.Name())

		return "", err
	}

	return tmp.Name(), nil
}

func probeDuration(ctx context.Context, ffprobe, path string) (time.Duration, error) {
	out, err := runTool(ctx, ffprobe,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	if err != nil {
		return 0, fmt.Errorf("viber_bm ffprobe: %w", err)
	}

	secs, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil || secs <= 0 {
		return 0, fmt.Errorf("viber_bm ffprobe: unreadable duration %q", strings.TrimSpace(string(out)))
	}

	return time.Duration(secs * float64(time.Second)), nil
}

// extractThumbnail grabs one frame, 1s in (or mid-clip for shorter videos) to
// skip the black lead-in frame most encoders emit.
func extractThumbnail(ctx context.Context, ffmpeg, path string, duration time.Duration) ([]byte, error) {
	at := min(time.Second, duration/2)

	out, err := runTool(ctx, ffmpeg,
		"-v", "error",
		"-ss", strconv.FormatFloat(at.Seconds(), 'f', 3, 64),
		"-i", path,
		"-frames:v", "1",
		"-vf", fmt.Sprintf("scale='min(%d,iw)':-2", thumbnailWidth),
		"-f", "image2",
		"-c:v", "mjpeg",
		"pipe:1",
	)
	if err != nil {
		return nil, fmt.Errorf("viber_bm ffmpeg thumbnail: %w", err)
	}

	if len(out) == 0 {
		return nil, errors.New("viber_bm ffmpeg thumbnail: no frame extracted")
	}

	return out, nil
}

func runTool(ctx context.Context, bin string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, truncateBody(stderr.Bytes()))
	}

	return stdout.Bytes(), nil
}

// isoDuration renders d as an ISO 8601 duration in whole seconds, rounded up
// so a sub-second clip is never reported as zero.
func isoDuration(d time.Duration) string {
	return fmt.Sprintf("PT%dS", int64(math.Ceil(d.Seconds())))
}
