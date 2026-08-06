package viber

import (
	"context"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

// sendKind is the Viber message type used to deliver an outbound attachment.
// https://developers.viber.com/docs/api/rest-bot-api/#message-types
type sendKind int

const (
	sendFile sendKind = iota
	sendPicture
	sendVideo
	sendGIF
)

func (k sendKind) String() string {
	switch k {
	case sendPicture:
		return "picture"
	case sendVideo:
		return "video"
	case sendGIF:
		return "url"
	case sendFile:
		return "file"
	default:
		return "file"
	}
}

// Viber platform caps. Oversized payloads are downgraded to a file instead of failing.
// https://developers.viber.com/docs/api/rest-bot-api/#sending-messages
const (
	maxVideoBytes  = 26 << 20
	maxFileBytes   = 50 << 20
	maxMediaURLLen = 2000
)

func classify(mimeType, fileName string) sendKind {
	m := normalizeMime(mimeType)
	if m == "" || m == "application/octet-stream" {
		if byExt := normalizeMime(mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))); byExt != "" {
			m = byExt
		}
	}

	switch {
	case m == "image/gif":
		return sendGIF
	case strings.HasPrefix(m, "image/"):
		return sendPicture
	case strings.HasPrefix(m, "video/"):
		return sendVideo
	default:
		return sendFile
	}
}

func normalizeMime(v string) string {
	if v == "" {
		return ""
	}
	parsed, _, err := mime.ParseMediaType(v)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(strings.Split(v, ";")[0]))
	}
	return strings.ToLower(parsed)
}

func (p *viberProvider) probeMedia(ctx context.Context, url string, mimeType *string, size *int64) {
	if url == "" || (*mimeType != "" && *size > 0) {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.DebugContext(ctx, "viber media probe failed", "url", url, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}
	if *mimeType == "" {
		*mimeType = resp.Header.Get("Content-Type")
	}
	if *size <= 0 && resp.ContentLength > 0 {
		*size = resp.ContentLength
	}
}
