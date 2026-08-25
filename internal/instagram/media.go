package instagram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

type syncedMedia struct {
	id       string
	mimeType string
	size     int64
}

func (p *instagramProvider) handleAttachments(ctx context.Context, gate *igmodel.InstagramGate, peers peerPair, attachments []Attachment, externalID, replyTo string) {
	for _, attach := range attachments {
		// Skip Instagram-only types that have no downloadable business content
		// before attempting any network call — these are handled separately below.
		switch attach.Type {
		case AttachmentTypeStoryMention:
			p.logger.Info("story mention not downloadable, skipping", "type", attach.Type)

			continue
		case AttachmentTypeIGReel, AttachmentTypeReel:
			p.logger.Warn("reel attachment has no downloadable business body, skipping", "type", attach.Type)

			continue
		case AttachmentTypeShare:
			p.logger.Warn("share attachment has no Messenger analog, skipping", "type", attach.Type)

			continue
		}

		if attach.Payload.URL == "" {
			continue
		}

		name := attachmentFileName(attach)

		media, err := p.downloadAndUpload(ctx, gate, attach.Payload.URL, name)
		if err != nil {
			p.logger.Error("failed to sync media", "url", attach.Payload.URL, "err", err)

			continue
		}

		if media.size <= 0 {
			media.size = 1
		}

		switch attach.Type {
		case "image":
			if _, err := p.coreMessengerFor(gate).SendImage(ctx, &sharedmodel.SendImageRequest{
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
				ExternalID:        externalID,
				ReplyToExternalID: replyTo,
			}); err != nil {
				p.logger.Error("failed to send image", "fileName", name, "err", err)
			}
		case "video", "audio", "file":
			if _, err := p.coreMessengerFor(gate).SendDocument(ctx, &sharedmodel.SendDocumentRequest{
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
				ExternalID:        externalID,
				ReplyToExternalID: replyTo,
			}); err != nil {
				p.logger.Error("failed to send document", "fileName", name, "err", err)
			}
		default:
			p.logger.Warn("unsupported attachment type, skipping", "type", attach.Type, "fileName", name)
		}
	}
}

func (p *instagramProvider) downloadAndUpload(ctx context.Context, gate *igmodel.InstagramGate, igURL, fileName string) (*syncedMedia, error) {
	// Guard the attacker-influenced payload URL before attaching the gate's
	// bearer token — refuse non-https/internal targets (SSRF, token exfiltration).
	if err := validateFetchURL(igURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, igURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+gate.IGToken)

	// httpClient is a guarded client that also blocks dialing non-public IPs.
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ig download: status %s", resp.Status)
	}

	mimeType := resp.Header.Get("Content-Type")
	size := resp.ContentLength

	uploaded, err := p.media.UploadFile(ctx, sharedmodel.UploadRequest{
		DomainID: gate.DomainID,
		Name:     fileName,
		MimeType: mimeType,
	}, io.LimitReader(resp.Body, maxInboundBytes))
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

	return fmt.Sprintf("ig_%s_%d%s", attach.Type, time.Now().Unix(), ext)
}
