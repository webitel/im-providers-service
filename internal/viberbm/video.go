package viberbm

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	pbstorage "github.com/webitel/im-providers-service/gen/go/storage"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// Infobip VIDEO content limits.
const (
	maxVideoBytes    = 200 << 20
	maxVideoDuration = 600 * time.Second
)

// sendVideo relies on storage for the thumbnail and duration Infobip requires;
// both are derived when the file is uploaded with generate_thumbnail.
func (p *viberBMProvider) sendVideo(ctx context.Context, g *vibbmmodel.ViberBMGate, to string, doc *sharedmodel.Document, req *sharedmodel.Message) (*sendResult, error) {
	fileID, err := strconv.ParseInt(doc.ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: storage file id %q", vibbmmodel.ErrVideoPreviewUnavailable, doc.ID)
	}

	res, err := p.links.GenerateFileLink(ctx, &pbstorage.GenerateFileLinkRequest{
		DomainId: g.DomainID,
		FileId:   fileID,
		Action:   "download",
		Source:   "file",
		Query:    map[string]string{"fetch_thumbnail": "true"},
		Metadata: true,
	})
	if err != nil {
		return nil, fmt.Errorf("viber_bm video preview: %w", err)
	}

	meta := res.GetMetadata()
	duration := time.Duration(meta.GetProperties().GetDuration()) * time.Millisecond

	switch {
	case meta.GetSize() > maxVideoBytes:
		return nil, vibbmmodel.ErrVideoTooLarge
	case meta.GetThumbnail().GetSize() <= 0 || duration <= 0:
		return nil, vibbmmodel.ErrVideoPreviewUnavailable
	case duration > maxVideoDuration:
		return nil, vibbmmodel.ErrVideoTooLong
	}

	thumbURL, err := absoluteLink(res)
	if err != nil {
		return nil, fmt.Errorf("viber_bm video thumbnail link: %w", err)
	}

	return p.api.SendVideo(ctx, g.BaseURL, g.APIKey, g.SenderName, to, videoContent{
		MediaURL:      doc.URL,
		MediaDuration: isoDuration(duration),
		ThumbnailURL:  thumbURL,
		Text:          req.Text,
	}, outboundMessageID(req.ID))
}

// absoluteLink mirrors thread-service link generation: storage answers with a
// path relative to base_url, and Infobip needs an absolute http(s) URL.
func absoluteLink(res *pbstorage.GenerateFileLinkResponse) (string, error) {
	base, err := url.Parse(res.GetBaseUrl())
	if err != nil {
		return "", fmt.Errorf("parse base url: %w", err)
	}

	rel, err := url.Parse(res.GetUrl())
	if err != nil {
		return "", fmt.Errorf("parse file url: %w", err)
	}

	full := base.ResolveReference(rel)
	if (full.Scheme != "https" && full.Scheme != "http") || full.Host == "" {
		return "", fmt.Errorf("storage link %q is not an absolute http(s) url", full.Redacted())
	}

	return full.String(), nil
}

// isoDuration renders d as an ISO 8601 duration in whole seconds, rounded up
// so a sub-second clip is never reported as zero.
func isoDuration(d time.Duration) string {
	return fmt.Sprintf("PT%dS", int64(math.Ceil(d.Seconds())))
}
