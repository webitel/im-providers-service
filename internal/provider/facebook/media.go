package facebook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/webitel/im-providers-service/internal/domain/model"
	"github.com/webitel/im-providers-service/internal/provider/facebook/payload"
)

type SyncMediaResponse struct {
	ID       string
	MimeType string
	Size     int64
}

func (p *facebookProvider) handleAttachments(ctx context.Context, gate *model.FacebookGate, from, to model.Peer, attachments []payload.InboundAttachment) {
	for _, attach := range attachments {
		fbURL := attach.Payload.URL
		if fbURL == "" {
			continue
		}

		fileName := p.generateFileName(attach)
		res, err := p.syncMedia(ctx, gate, fbURL, fileName)
		if err != nil {
			p.logger.Error("failed to sync media", "url", fbURL, "err", err)
			continue
		}

		if res.Size <= 0 {
			res.Size = 1
		}

		switch attach.Type {
		case "image":
			if _, err := p.messenger.SendImage(ctx, &model.SendImageRequest{
				DomainID: gate.DomainID,
				From:     from,
				To:       to,
				Image: model.ImageRequest{
					Images: []*model.Image{{
						ID:       res.ID,
						FileName: fileName,
						MimeType: res.MimeType,
					}},
				},
			}); err != nil {
				p.logger.Error("failed to send image", "fileName", fileName, "err", err)
			}
		case "video", "audio", "file":
			if _, err := p.messenger.SendDocument(ctx, &model.SendDocumentRequest{
				DomainID: gate.DomainID,
				From:     from,
				To:       to,
				Document: model.DocumentRequest{
					Documents: []*model.Document{{
						ID:       res.ID,
						FileName: fileName,
						MimeType: res.MimeType,
						Size:     res.Size,
					}},
				},
			}); err != nil {
				p.logger.Error("failed to send document", "fileName", fileName, "err", err)
			}
		}
	}
}

func (p *facebookProvider) syncMedia(ctx context.Context, gate *model.FacebookGate, fbURL, fileName string) (*SyncMediaResponse, error) {
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
		return nil, fmt.Errorf("fb download failed with status: %s", resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	size := resp.ContentLength

	res, err := p.media.UploadFile(ctx, model.UploadRequest{
		DomainID: gate.DomainID,
		Name:     fileName,
		MimeType: contentType,
	}, resp.Body)
	if err != nil {
		return nil, err
	}

	return &SyncMediaResponse{
		ID:       res.ID,
		MimeType: contentType,
		Size:     size,
	}, nil
}

func (p *facebookProvider) generateFileName(attach payload.InboundAttachment) string {
	name := attach.Payload.Title
	if name == "" {
		name = attach.Payload.Name
	}
	if name != "" {
		return name
	}

	ext := ".bin"
	switch attach.Type {
	case "image":
		ext = ".jpg"
	case "video":
		ext = ".mp4"
	case "audio":
		ext = ".mp3"
	}
	return fmt.Sprintf("fb_%s_%d%s", attach.Type, time.Now().Unix(), ext)
}
