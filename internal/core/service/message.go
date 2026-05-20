package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
)

var _ Messenger = (*messageService)(nil)

type Messenger interface {
	SendText(ctx context.Context, in *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error)
	SendImage(ctx context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error)
	SendDocument(ctx context.Context, in *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error)
}

type messageService struct {
	logger    *slog.Logger
	gatewayer *imgateway.Client
}

func NewMessageService(logger *slog.Logger, threadClient *imgateway.Client) Messenger {
	return &messageService{
		logger:    logger.With("pkg", "service.messenger"),
		gatewayer: threadClient,
	}
}

func (m *messageService) SendText(ctx context.Context, in *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error) {
	m.logger.Info("dispatching text message to gateway", "from_sub", in.From.Sub)

	resp, err := m.gatewayer.SendText(ctx, &gatewayv1.SendTextRequest{
		To: &gatewayv1.Peer{
			Kind: &gatewayv1.Peer_Contact{
				Contact: &gatewayv1.PeerIdentity{
					Sub: in.To.Sub,
					Iss: in.To.Iss,
				},
			},
		},
		Body: in.Body,
	})
	if err != nil {
		m.logger.Error("failed to send text message", "error", err)
		return nil, err
	}

	return &sharedmodel.SendTextResponse{To: in.To, ID: m.parseUUID(resp.GetId())}, nil
}

func (m *messageService) SendImage(ctx context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) {
	resp, err := m.gatewayer.SendImage(ctx, &gatewayv1.SendImageRequest{
		To: &gatewayv1.Peer{
			Kind: &gatewayv1.Peer_Contact{
				Contact: &gatewayv1.PeerIdentity{
					Sub: in.To.Sub,
					Iss: in.To.Iss,
				},
			},
		},
		Body:   in.Image.Body,
		Images: m.mapImages(in.Image.Images),
	})
	if err != nil {
		m.logger.Error("failed to send image message", "error", err)
		return nil, err
	}

	return &sharedmodel.SendImageResponse{To: in.To, ID: m.parseUUID(resp.GetId())}, nil
}

func (m *messageService) SendDocument(ctx context.Context, in *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error) {
	resp, err := m.gatewayer.SendDocument(ctx, &gatewayv1.SendDocumentRequest{
		To: &gatewayv1.Peer{
			Kind: &gatewayv1.Peer_Contact{
				Contact: &gatewayv1.PeerIdentity{
					Sub: in.To.Sub,
					Iss: in.To.Iss,
				},
			},
		},
		Body:      in.Document.Body,
		Documents: m.mapDocuments(in.Document.Documents),
	})
	if err != nil {
		m.logger.Error("failed to send document message", "error", err)
		return nil, err
	}

	return &sharedmodel.SendDocumentResponse{To: in.To, ID: m.parseUUID(resp.GetId())}, nil
}

// --- Helpers ---

func (m *messageService) mapImages(src []*sharedmodel.Image) []*gatewayv1.ImageInput {
	res := make([]*gatewayv1.ImageInput, 0, len(src))
	for _, img := range src {
		if img == nil {
			continue
		}
		res = append(res, &gatewayv1.ImageInput{
			Id: img.ID, Name: img.FileName, Link: img.URL, MimeType: img.MimeType,
		})
	}
	return res
}

func (m *messageService) mapDocuments(src []*sharedmodel.Document) []*gatewayv1.DocumentInput {
	res := make([]*gatewayv1.DocumentInput, 0, len(src))
	for _, doc := range src {
		if doc == nil {
			continue
		}
		size := doc.Size
		res = append(res, &gatewayv1.DocumentInput{
			Id: doc.ID, Url: doc.URL, FileName: doc.FileName, MimeType: doc.MimeType, SizeBytes: &size,
		})
	}
	return res
}

func (m *messageService) parseUUID(id string) uuid.UUID {
	if id == "" {
		return uuid.Nil
	}
	res, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil
	}
	return res
}
