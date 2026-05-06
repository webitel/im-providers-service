package service

import (
	"context"
	"fmt"

	client "github.com/webitel/im-providers-service/infra/client/grpc"
	"github.com/webitel/im-providers-service/internal/domain/model"
)

type messengerAuthMiddleware struct {
	next Messenger
}

func NewMessengerAuthMiddleware(next Messenger) Messenger {
	return &messengerAuthMiddleware{next: next}
}

func (m *messengerAuthMiddleware) withIdentity(ctx context.Context, dc int64, sub string) context.Context {
	return client.WithIdentity(ctx, client.StringIdentity(fmt.Sprintf("%d.%s", dc, sub)))
}

func (m *messengerAuthMiddleware) SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error) {
	return m.next.SendText(m.withIdentity(ctx, in.DomainID, in.From.Sub), in)
}

func (m *messengerAuthMiddleware) SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error) {
	return m.next.SendImage(m.withIdentity(ctx, in.DomainID, in.From.Sub), in)
}

func (m *messengerAuthMiddleware) SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error) {
	return m.next.SendDocument(m.withIdentity(ctx, in.DomainID, in.From.Sub), in)
}
