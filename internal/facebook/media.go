package facebook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
)

type syncedMedia struct {
	id       string
	mimeType string
	size     int64
}

func (p *facebookProvider) handleAttachments(ctx context.Context, gate *fbmodel.FacebookGate, peers peerPair, attachments []Attachment) {
	for _, attach := range attachments {
		if attach.Payload.URL == "" {
			continue
		}

		name := attachmentFileName(attach)
		media, err := p.downloadAndUpload(ctx, gate, attach.Payload.URL, name)
		if err != nil {
			p.logger.Error("failed to sync media", "url", attach.Payload.URL, semconv.ErrorKey, err)
			continue
		}

		if media.size <= 0 {
			media.size = 1
		}

		switch attach.Type {
		case "image":
			if _, err := p.messenger.SendImage(ctx, &sharedmodel.SendImageRequest{
				DomainID: gate.DomainID,
				From:     peers.from,
				To:       peers.to,
				Image: sharedmodel.ImageRequest{
					Images: []*sharedmodel.Image{{
						ID:       media.id,
						FileName: name,
						MimeType: media.mimeType,
					}},
				},
			}); err != nil {
				p.logger.Error("failed to send image", "fileName", name, semconv.ErrorKey, err)
			}
		case "video", "audio", "file":
			if _, err := p.messenger.SendDocument(ctx, &sharedmodel.SendDocumentRequest{
				DomainID: gate.DomainID,
				From:     peers.from,
				To:       peers.to,
				Document: sharedmodel.DocumentRequest{
					Documents: []*sharedmodel.Document{{
						ID:       media.id,
						FileName: name,
						MimeType: media.mimeType,
						Size:     media.size,
					}},
				},
			}); err != nil {
				p.logger.Error("failed to send document", "fileName", name, semconv.ErrorKey, err)
			}
		default:
			p.logger.Warn("unsupported attachment type, skipping", "type", attach.Type, "fileName", name)
		}
	}
}

func (p *facebookProvider) downloadAndUpload(ctx context.Context, gate *fbmodel.FacebookGate, fbURL, fileName string) (*syncedMedia, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fbURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+gate.PageToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fb download: status %s", resp.Status)
	}

	mimeType := resp.Header.Get("Content-Type")
	size := resp.ContentLength

	uploaded, err := p.media.UploadFile(ctx, sharedmodel.UploadRequest{
		DomainID: gate.DomainID,
		Name:     fileName,
		MimeType: mimeType,
	}, resp.Body)
	if err != nil {
		return nil, err
	}

	return &syncedMedia{id: uploaded.ID, mimeType: mimeType, size: size}, nil
}

func attachmentFileName(attach Attachment) string {
	if attach.Payload.Title != "" {
		return attach.Payload.Title
	}
	if attach.Payload.Name != "" {
		return attach.Payload.Name
	}

	ext := map[string]string{
		"image": ".jpg",
		"video": ".mp4",
		"audio": ".mp3",
	}[attach.Type]
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("fb_%s_%d%s", attach.Type, time.Now().Unix(), ext)
}
