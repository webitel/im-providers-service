package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	gatewayv1 "github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/webitel-go-kit/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ Messenger = (*messageService)(nil)

type Messenger interface {
	SendText(ctx context.Context, in *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error)
	SendImage(ctx context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error)
	SendDocument(ctx context.Context, in *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error)
	SendLocation(ctx context.Context, in *sharedmodel.SendLocationRequest) (*sharedmodel.SendResponse, error)
	SendContact(ctx context.Context, in *sharedmodel.SendContactRequest) (*sharedmodel.SendResponse, error)
	SendInteractiveCallback(ctx context.Context, in *sharedmodel.SendInteractiveCallbackRequest) error
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

func transformDomainPeerIntoPB(peer sharedmodel.Peer) *gatewayv1.Peer {
	return &gatewayv1.Peer{
		Kind: &gatewayv1.Peer_Contact{
			Contact: &gatewayv1.PeerIdentity{
				Sub: peer.Sub,
				Iss: peer.Iss,
			},
		},
	}
}

func (m *messageService) SendLocation(ctx context.Context, in *sharedmodel.SendLocationRequest) (*sharedmodel.SendResponse, error) {
	resp, err := m.gatewayer.SendLocation(ctx, &gatewayv1.SendLocationRequest{
		To:        transformDomainPeerIntoPB(in.To),
		Latitude:  in.Latitude,
		Longitude: in.Longitude,
		Name:      in.Name,
		Address:   in.Address,
		SendId:    in.ExternalID,
	})

	if err != nil {
		return nil, errors.Wrap(err, errors.WithID("service.message.send_location"))
	}

	return &sharedmodel.SendResponse{
		ID: m.parseUUID(resp.GetId()),
		To: in.To,
	}, nil
}

func (m *messageService) SendContact(ctx context.Context, in *sharedmodel.SendContactRequest) (*sharedmodel.SendResponse, error) {
	contactMatadata, err := structpb.NewStruct(in.Metadata)
	if err != nil {
		return nil, errors.InvalidArgument("converting model metadata to structb", errors.WithCause(err), errors.WithID("service.message.send_contact"))
	}

	resp, err := m.gatewayer.SendContact(ctx, &gatewayv1.SendContactRequest{
		To:          transformDomainPeerIntoPB(in.To),
		Name:        in.Name,
		Email:       in.Email,
		PhoneNumber: in.PhoneNumber,
		Metadata:    contactMatadata,
		SendId:      "",
	})

	if err != nil {
		return nil, errors.Wrap(err, errors.WithID("service.message.send_contact"))
	}

	return &sharedmodel.SendResponse{
		ID: m.parseUUID(resp.GetId()),
		To: in.To,
	}, nil
}

// SendImage forwards received images to the core gateway.
// The gateway SendImage rpc was removed in favor of consolidating on SendDocument,
// so inbound images are delivered to the core as document attachments.
func (m *messageService) SendImage(ctx context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) {
	m.logger.Info("dispatching image message to gateway as document",
		"from_sub", in.From.Sub,
		"images_count", len(in.Image.Images),
	)

	resp, err := m.gatewayer.SendDocument(ctx, &gatewayv1.SendDocumentRequest{
		To: &gatewayv1.Peer{
			Kind: &gatewayv1.Peer_Contact{
				Contact: &gatewayv1.PeerIdentity{
					Sub: in.To.Sub,
					Iss: in.To.Iss,
				},
			},
		},
		Body:      in.Image.Body,
		Documents: m.mapImagesAsDocuments(in.Image.Images),
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

// mapImagesAsDocuments converts inbound image attachments into gateway document inputs,
// since received images are forwarded to the core via the SendDocument rpc.
func (m *messageService) mapImagesAsDocuments(src []*sharedmodel.Image) []*gatewayv1.DocumentInput {
	res := make([]*gatewayv1.DocumentInput, 0, len(src))
	for _, img := range src {
		if img == nil {
			continue
		}
		res = append(res, &gatewayv1.DocumentInput{
			Id: img.ID, Url: img.URL, FileName: img.FileName, MimeType: img.MimeType,
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

func (m *messageService) SendInteractiveCallback(ctx context.Context, in *sharedmodel.SendInteractiveCallbackRequest) error {
	_, err := m.gatewayer.SendInteractiveCallback(ctx, &gatewayv1.InteractiveCallbackRequest{
		InReplyTo:    in.InReplyTo,
		ButtonCode:   in.ButtonCode,
		CallbackData: in.CallbackData,
	})
	if err != nil {
		m.logger.Error("failed to send interactive callback", "error", err)
		return err
	}
	return nil
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
