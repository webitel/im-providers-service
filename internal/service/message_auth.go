package service

import (
	"context"
	"fmt"

	client "github.com/webitel/im-providers-service/infra/client/grpc"
	"github.com/webitel/im-providers-service/internal/domain/model"
)

type gateIdentity string

func (g gateIdentity) Identity() string { return string(g) }

type messengerAuthMiddleware struct {
	next Messenger
}

func NewMessengerAuthMiddleware(next Messenger) Messenger {
	return &messengerAuthMiddleware{next: next}
}

func (m *messengerAuthMiddleware) withIdentity(ctx context.Context, dc int64, sub string) context.Context {
	id := fmt.Sprintf("%d.%s", dc, sub)
	return client.WithIdentity(ctx, gateIdentity(id))
}

func (m *messengerAuthMiddleware) SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error) {
	ctx = m.withIdentity(ctx, in.DomainID, in.From.Sub)
	return m.next.SendText(ctx, in)
}

func (m *messengerAuthMiddleware) SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error) {
	ctx = m.withIdentity(ctx, in.DomainID, in.From.Sub)
	return m.next.SendImage(ctx, in)
}

func (m *messengerAuthMiddleware) SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error) {
	ctx = m.withIdentity(ctx, in.DomainID, in.From.Sub)
	return m.next.SendDocument(ctx, in)
}
