package viberbm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

type syncedMedia struct {
	id       string
	mimeType string
	size     int64
	isImage  bool
}

// downloadInboundMedia fetches an InfoBip media URL and uploads it to storage.
// Authorization on the GET is undocumented, so it tries authenticated first and
// falls back unauthenticated on 401/403.
// https://www.infobip.com/docs/api/channels/viber/viber-business-messages/inbound-message/receive-viber-business-message
func (p *viberBMProvider) downloadInboundMedia(ctx context.Context, gate *vibbmmodel.ViberBMGate, mediaURL, fileName string) (*syncedMedia, error) {
	if err := validateFetchURL(mediaURL); err != nil {
		return nil, err
	}

	resp, err := p.fetchMedia(ctx, gate.APIKey, mediaURL, true)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		_ = resp.Body.Close()

		resp, err = p.fetchMedia(ctx, gate.APIKey, mediaURL, false)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("viber_bm media download: status %s", resp.Status)
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

	if size <= 0 {
		size = 1
	}

	return &syncedMedia{
		id:       uploaded.ID,
		mimeType: mimeType,
		size:     size,
		// Image vs document is branched on content-type / extension.
		isImage: isImageMedia(mimeType, fileName),
	}, nil
}

func (p *viberBMProvider) fetchMedia(ctx context.Context, apiKey, mediaURL string, authenticated bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, err
	}

	if authenticated {
		req.Header.Set("Authorization", authScheme+" "+apiKey)
	}

	return p.linkClient.Do(req)
}

func isImageMedia(mimeType, fileName string) bool {
	if strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return true
	}

	lower := strings.ToLower(fileName)
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".gif", ".webp"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	return false
}
