package service

import (
	"context"
	"fmt"

	client "github.com/webitel/im-providers-service/infra/client/grpc"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

type messengerAuthMiddleware struct {
	Messenger
}

func NewMessengerAuthMiddleware(next Messenger) Messenger {
	return &messengerAuthMiddleware{Messenger: next}
}

func (m *messengerAuthMiddleware) withIdentity(ctx context.Context, dc int64, sub string) context.Context {
	return client.WithIdentity(ctx, client.StringIdentity(fmt.Sprintf("%d.%s", dc, sub)))
}

<<<<<<< HEAD:internal/core/service/message_auth.go
func (m *messengerAuthMiddleware) SendText(ctx context.Context, in *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error) {
	return m.next.SendText(m.withIdentity(ctx, in.DomainID, in.From.Sub), in)
}

func (m *messengerAuthMiddleware) SendImage(ctx context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) {
	return m.next.SendImage(m.withIdentity(ctx, in.DomainID, in.From.Sub), in)
}

func (m *messengerAuthMiddleware) SendDocument(ctx context.Context, in *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error) {
	return m.next.SendDocument(m.withIdentity(ctx, in.DomainID, in.From.Sub), in)
}
