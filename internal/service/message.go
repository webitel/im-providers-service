package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/internal/domain/model"
)

var _ Messenger = (*MessageService)(nil)

type Messenger interface {
	SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error)
	SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error)
	SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error)
}

type MessageService struct {
	logger    *slog.Logger
	gatewayer *imgateway.Client
}

func NewMessageService(logger *slog.Logger, threadClient *imgateway.Client) *MessageService {
	return &MessageService{
		logger:    logger.With("pkg", "service.messenger"),
		gatewayer: threadClient,
	}
}

// SendText handles plain text message delivery from providers to the internal gateway.
func (m *MessageService) SendText(ctx context.Context, in *model.SendTextRequest) (*model.SendTextResponse, error) {
	m.logger.Info("dispatching text message to gateway",
		"from_sub", in.From.Sub,
		"issuer", in.From.Iss,
		"body_len", len(in.Body),
	)

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
		m.logger.Error("gateway text dispatch failed", "err", err, "from_sub", in.From.Sub)
		return nil, err
	}

	return &model.SendTextResponse{
		To: in.To,
		ID: m.parseUUID(resp.GetId()),
	}, nil
}

// SendImage handles image gallery delivery.
func (m *MessageService) SendImage(ctx context.Context, in *model.SendImageRequest) (*model.SendImageResponse, error) {
	m.logger.Info("dispatching image message to gateway",
		"from_sub", in.From.Sub,
		"images_count", len(in.Image.Images),
	)

	// Removed the intermediate ImageRequest wrapper as it was undefined in proto generation.
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
		m.logger.Error("gateway image dispatch failed", "err", err, "from_sub", in.From.Sub)
		return nil, err
	}

	return &model.SendImageResponse{
		To: in.To,
		ID: m.parseUUID(resp.GetId()),
	}, nil
}

// SendDocument handles file/attachment delivery.
func (m *MessageService) SendDocument(ctx context.Context, in *model.SendDocumentRequest) (*model.SendDocumentResponse, error) {
	m.logger.Info("dispatching document message to gateway",
		"from_sub", in.From.Sub,
		"docs_count", len(in.Document.Documents),
	)

	// [FIX]: Mapping documents directly to the SendDocumentRequest.
	// Removed the intermediate DocumentRequest wrapper.
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
		m.logger.Error("gateway document dispatch failed", "err", err, "from_sub", in.From.Sub)
		return nil, err
	}

	return &model.SendDocumentResponse{
		To: in.To,
		ID: m.parseUUID(resp.GetId()),
	}, nil
}

// --- Internal Helpers & Mappers ---

func (m *MessageService) mapImages(src []*model.Image) []*gatewayv1.ImageInput {
	res := make([]*gatewayv1.ImageInput, 0, len(src))
	for _, img := range src {
		if img == nil {
			continue
		}
		res = append(res, &gatewayv1.ImageInput{
			Id:       img.ID,
			Name:     img.FileName,
			Link:     img.URL,
			MimeType: img.MimeType,
		})
	}
	return res
}

func (m *MessageService) mapDocuments(src []*model.Document) []*gatewayv1.DocumentInput {
	res := make([]*gatewayv1.DocumentInput, 0, len(src))
	for _, doc := range src {
		if doc == nil {
			continue
		}
		// Explicitly handle the size pointer for gRPC safety
		size := doc.Size
		res = append(res, &gatewayv1.DocumentInput{
			Id:        doc.ID,
			Url:       doc.URL,
			FileName:  doc.FileName,
			MimeType:  doc.MimeType,
			SizeBytes: &size,
		})
	}
	return res
}

func (m *MessageService) parseUUID(id string) uuid.UUID {
	if id == "" {
		return uuid.Nil
	}
	res, err := uuid.Parse(id)
	if err != nil {
		m.logger.Warn("invalid uuid in gateway response", slog.String("raw_id", id))
		return uuid.Nil
	}
	return res
}
