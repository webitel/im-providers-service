package facebook

import (
	"context"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

func (p *facebookProvider) SendText(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	return p.api.SendText(ctx, g.PageToken, req.To.Sub, req.Text)
}

func (p *facebookProvider) SendImage(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaImage, firstURL(req.Images))
}

func (p *facebookProvider) SendDocument(ctx context.Context, req *sharedmodel.Message) (*sharedmodel.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaFile, firstURL(req.Documents))
}

type urlGetter interface {
	GetURL() string
}

// firstURL returns the URL of the first element, or "" if the slice is empty.
func firstURL[T urlGetter](items []T) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].GetURL()
}
