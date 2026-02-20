package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/service/dto"
)

var _ Messenger = (*MessageService)(nil)

type Messenger interface {
	SendText(ctx context.Context, in *dto.SendTextRequest) (*dto.SendTextResponse, error)
	SendImage(ctx context.Context, in *dto.SendImageRequest) (*dto.SendImageResponse, error)
	SendDocument(ctx context.Context, in *dto.SendDocumentRequest) (*dto.SendDocumentResponse, error)
}

type MessageService struct {
	logger    *slog.Logger
	gatewayer *imgateway.Client
}

func NewMessageService(logger *slog.Logger, threadClient *imgateway.Client) *MessageService {
	return &MessageService{
		logger:    logger,
		gatewayer: threadClient,
	}
}

// SendText handles plain text message delivery
func (m *MessageService) SendText(ctx context.Context, in *dto.SendTextRequest) (*dto.SendTextResponse, error) {
	// [TRANSPORT_LOGIC] Handling outgoing message delivery via Gateway
	resp, err := m.gatewayer.SendText(ctx, &gatewayv1.SendTextRequest{
		To: &gatewayv1.Peer{
			Kind: &gatewayv1.Peer_Contact{
				Contact: &gatewayv1.PeerIdentity{
					Sub: in.From.ID.String(),
					Iss: in.From.Issuer,
				},
			},
		},
		Body: in.Body,
	})
	if err != nil {
		return nil, err
	}

	return &dto.SendTextResponse{To: in.To, ID: m.parseUUID(resp.GetId())}, nil
}

// SendImage handles image gallery delivery
func (m *MessageService) SendImage(ctx context.Context, in *dto.SendImageRequest) (*dto.SendImageResponse, error) {
	resp, err := m.gatewayer.SendImage(ctx, &gatewayv1.SendImageRequest{
		To: &gatewayv1.Peer{
			Kind: &gatewayv1.Peer_Contact{
				Contact: &gatewayv1.PeerIdentity{
					Sub: in.From.ID.String(),
					Iss: in.From.Issuer,
				},
			},
		},
		Image: &gatewayv1.ImageRequest{
			Body:   in.Image.Body,
			Images: m.mapImages(in.Image.Images),
		},
	})
	if err != nil {
		return nil, err
	}

	return &dto.SendImageResponse{To: in.To, ID: m.parseUUID(resp.GetId())}, nil
}

// SendDocument handles file/attachment delivery
func (m *MessageService) SendDocument(ctx context.Context, in *dto.SendDocumentRequest) (*dto.SendDocumentResponse, error) {
	resp, err := m.gatewayer.SendFile(ctx, &gatewayv1.SendDocumentRequest{
		To: &gatewayv1.Peer{
			Kind: &gatewayv1.Peer_Contact{
				Contact: &gatewayv1.PeerIdentity{
					Sub: in.From.ID.String(),
					Iss: in.From.Issuer,
				},
			},
		},
		Document: &gatewayv1.DocumentRequest{
			Body:      in.Document.Body,
			Documents: m.mapDocuments(in.Document.Documents),
		},
	})
	if err != nil {
		return nil, err
	}

	return &dto.SendDocumentResponse{To: in.To, ID: m.parseUUID(resp.GetId())}, nil
}

// --- Internal Helpers & Mappers ---

func coalesceString(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

func (m *MessageService) mapImages(src []*dto.Image) []*gatewayv1.ImageInput {
	res := make([]*gatewayv1.ImageInput, 0, len(src))
	for _, img := range src {
		if img == nil {
			continue
		}
		res = append(res, &gatewayv1.ImageInput{
			Id:       fmt.Sprintf("%d", img.ID),
			Name:     img.Name,
			Link:     img.URL,
			MimeType: img.MimeType,
		})
	}
	return res
}

func (m *MessageService) mapDocuments(src []*dto.Document) []*gatewayv1.DocumentInput {
	res := make([]*gatewayv1.DocumentInput, 0, len(src))
	for _, doc := range src {
		if doc == nil {
			continue
		}
		size := doc.Size
		res = append(res, &gatewayv1.DocumentInput{
			Id:        fmt.Sprintf("%d", doc.ID),
			Url:       doc.URL,
			FileName:  doc.Name,
			MimeType:  doc.MimeType,
			SizeBytes: &size,
		})
	}
	return res
}

func (m *MessageService) parseUUID(id string) uuid.UUID {
	res, err := uuid.Parse(id)
	if err != nil {
		m.logger.Warn("invalid uuid in response", slog.String("raw_id", id))
		return uuid.Nil
	}
	return res
}
