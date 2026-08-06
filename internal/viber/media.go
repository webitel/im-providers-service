package viber

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

const maxInboundBytes = 25 << 20

type syncedMedia struct {
	id       string
	name     string
	mimeType string
	size     int64
}

func (p *viberProvider) handleMedia(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, msg *inboundMessage, externalID string) {
	if msg.Media == "" {
		return
	}

	media, err := p.syncMedia(ctx, gate.DomainID, msg)
	if err != nil {
		p.logger.Error("failed to sync viber media", "url", msg.Media, "err", err)
		return
	}

	p.forwardMedia(ctx, gate, peers, media, externalID)
}

func (p *viberProvider) forwardMedia(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, media *syncedMedia, externalID string) {
	if strings.HasPrefix(media.mimeType, "image/") {
		if _, err := p.messenger.SendImage(ctx, &sharedmodel.SendImageRequest{
			DomainID: gate.DomainID,
			From:     peers.from,
			To:       peers.to,
			Image: sharedmodel.ImageRequest{
				Images: []*sharedmodel.Image{{
					ID:       media.id,
					FileName: media.name,
					MimeType: media.mimeType,
					Size:     media.size,
				}},
			},
			ExternalID: externalID,
		}); err != nil {
			p.logger.Error("failed to send image", "fileName", media.name, "err", err)
		}

		return
	}

	if _, err := p.messenger.SendDocument(ctx, &sharedmodel.SendDocumentRequest{
		DomainID: gate.DomainID,
		From:     peers.from,
		To:       peers.to,
		Document: sharedmodel.DocumentRequest{
			Documents: []*sharedmodel.Document{{
				ID:       media.id,
				FileName: media.name,
				MimeType: media.mimeType,
				Size:     media.size,
			}},
		},
		ExternalID: externalID,
	}); err != nil {
		p.logger.Error("failed to send document", "fileName", media.name, "err", err)
	}
}

// syncMedia downloads a webhook-supplied URL and stores it. The URL is
// untrusted, so it always goes through validateFetchURL and linkClient, which
// refuses to dial non-public addresses and only follows https redirects — a
// plain client here would turn any inbound message into a request-forgery
// primitive with the response body readable from the chat.
func (p *viberProvider) syncMedia(ctx context.Context, domainID int64, msg *inboundMessage) (*syncedMedia, error) {
	if err := validateFetchURL(msg.Media); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, msg.Media, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.linkClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("viber download: status %s", resp.Status)
	}

	mimeType := resolveMime(msg, resp.Header.Get("Content-Type"))
	name := viberFileName(msg, mimeType)

	uploaded, err := p.media.UploadFile(ctx, sharedmodel.UploadRequest{
		DomainID: domainID,
		Name:     name,
		MimeType: mimeType,
	}, io.LimitReader(resp.Body, maxInboundBytes))
	if err != nil {
		return nil, err
	}

	size := msg.byteSize()
	if size <= 0 {
		size = resp.ContentLength
	}
	if size <= 0 {
		size = 1
	}

	return &syncedMedia{id: uploaded.ID, name: name, mimeType: mimeType, size: size}, nil
}

var audioMimeByExt = map[string]string{
	".m4a":  "audio/mp4",
	".mp3":  "audio/mpeg",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/ogg",
	".aac":  "audio/aac",
	".wav":  "audio/wav",
	".amr":  "audio/amr",
}

var preferredExt = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
	"audio/mp4":       ".m4a",
	"audio/mpeg":      ".mp3",
	"audio/ogg":       ".ogg",
	"audio/aac":       ".aac",
	"audio/wav":       ".wav",
	"audio/amr":       ".amr",
}

func resolveMime(msg *inboundMessage, headerContentType string) string {
	if m := normalizeMime(headerContentType); m != "" && m != "application/octet-stream" {
		return m
	}

	if ext := strings.ToLower(filepath.Ext(msg.FileName)); ext != "" {
		if m, ok := audioMimeByExt[ext]; ok {
			return m
		}
		if m := normalizeMime(mime.TypeByExtension(ext)); m != "" {
			return m
		}
	}

	switch msg.Type {
	case "picture":
		return "image/jpeg"
	case "video":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

func viberFileName(msg *inboundMessage, mimeType string) string {
	if msg.FileName != "" {
		return msg.FileName
	}

	ext := preferredExt[mimeType]
	if ext == "" {
		if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
			ext = exts[0]
		}
	}
	if ext == "" {
		ext = ".bin"
	}

	kind := msg.Type
	if kind == "" {
		kind = "file"
	}

	return fmt.Sprintf("viber_%s_%d%s", kind, time.Now().Unix(), ext)
}
