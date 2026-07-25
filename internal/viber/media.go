package viber

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibmodel "github.com/webitel/im-providers-service/internal/viber/model"
)

type syncedMedia struct {
	id       string
	mimeType string
	size     int64
}

// handleMedia downloads the Viber-hosted media, re-uploads it to Webitel storage,
// and forwards the stored file to the core as an image or document.
func (p *viberProvider) handleMedia(ctx context.Context, gate *vibmodel.ViberGate, peers peerPair, msg *inboundMessage, externalID string, kind mediaKind) {
	if msg.Media == "" {
		return
	}

	name := viberFileName(msg, kind)
	media, err := p.downloadAndUpload(ctx, gate.DomainID, msg.Media, name)
	if err != nil {
		p.logger.Error("failed to sync viber media", "url", msg.Media, "err", err)
		return
	}

	size := msg.Size
	if size <= 0 {
		size = media.size
	}
	if size <= 0 {
		size = 1
	}

	switch kind {
	case mediaImage:
		if _, err := p.messenger.SendImage(ctx, &sharedmodel.SendImageRequest{
			DomainID: gate.DomainID,
			From:     peers.from,
			To:       peers.to,
			Image: sharedmodel.ImageRequest{
				Images: []*sharedmodel.Image{{
					ID:       media.id,
					FileName: name,
					MimeType: media.mimeType,
					Size:     size,
				}},
			},
			ExternalID: externalID,
		}); err != nil {
			p.logger.Error("failed to send image", "fileName", name, "err", err)
		}
	case mediaDocument:
		if _, err := p.messenger.SendDocument(ctx, &sharedmodel.SendDocumentRequest{
			DomainID: gate.DomainID,
			From:     peers.from,
			To:       peers.to,
			Document: sharedmodel.DocumentRequest{
				Documents: []*sharedmodel.Document{{
					ID:       media.id,
					FileName: name,
					MimeType: media.mimeType,
					Size:     size,
				}},
			},
			ExternalID: externalID,
		}); err != nil {
			p.logger.Error("failed to send document", "fileName", name, "err", err)
		}
	}
}

// downloadAndUpload streams the media from the (public, temporary) Viber URL into
// Webitel storage. No Authorization header is required — Viber media URLs are public.
func (p *viberProvider) downloadAndUpload(ctx context.Context, domainID int64, mediaURL, fileName string) (*syncedMedia, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("viber download: status %s", resp.Status)
	}

	mimeType := resp.Header.Get("Content-Type")
	size := resp.ContentLength

	uploaded, err := p.media.UploadFile(ctx, sharedmodel.UploadRequest{
		DomainID: domainID,
		Name:     fileName,
		MimeType: mimeType,
	}, resp.Body)
	if err != nil {
		return nil, err
	}

	return &syncedMedia{id: uploaded.ID, mimeType: mimeType, size: size}, nil
}

func viberFileName(msg *inboundMessage, kind mediaKind) string {
	if msg.FileName != "" {
		return msg.FileName
	}
	ext := ".bin"
	switch {
	case kind == mediaImage:
		ext = ".jpg"
	case msg.Type == "video":
		ext = ".mp4"
	}
	return fmt.Sprintf("viber_%s_%d%s", msg.Type, time.Now().Unix(), ext)
}
