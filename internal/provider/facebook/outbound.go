package facebook

import (
	"context"

	"github.com/webitel/im-providers-service/internal/domain/model"
)

func (p *facebookProvider) SendText(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	return p.api.SendText(ctx, g.PageToken, req.To.Sub, req.Text)
}

func (p *facebookProvider) SendImage(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	u := ""
	if len(req.Images) > 0 {
		u = req.Images[0].URL
	}
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaImage, u)
}

func (p *facebookProvider) SendDocument(ctx context.Context, req *model.Message) (*model.MessageResponse, error) {
	g, err := p.fetchGate(ctx, req.GateID)
	if err != nil {
		return nil, err
	}
	u := ""
	if len(req.Documents) > 0 {
		u = req.Documents[0].URL
	}
	return p.api.SendMedia(ctx, g.PageToken, req.To.Sub, MediaFile, u)
}
