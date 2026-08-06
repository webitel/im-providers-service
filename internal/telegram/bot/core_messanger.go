package bot

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

// viaHeader identifies which gate a message forwarded to im-gateway-service
// belongs to, so the same external contact sub can be disambiguated across
// multiple Telegram bots.
const viaHeader = "x-webitel-via"

// CoreMessanger is the subset of sharedsvc.Messenger the Telegram provider
// forwards inbound webhook updates through.
type CoreMessanger interface {
	SendText(ctx context.Context, in *coremodel.SendTextRequest) (*coremodel.SendTextResponse, error)
	SendImage(ctx context.Context, in *coremodel.SendImageRequest) (*coremodel.SendImageResponse, error)
	SendDocument(ctx context.Context, in *coremodel.SendDocumentRequest) (*coremodel.SendDocumentResponse, error)
	SendLocation(ctx context.Context, in *coremodel.SendLocationRequest) (*coremodel.SendResponse, error)
	SendContact(ctx context.Context, in *coremodel.SendContactRequest) (*coremodel.SendResponse, error)
	SendInteractiveCallback(ctx context.Context, in *coremodel.SendInteractiveCallbackRequest) error
}

// viaCoreMessanger decorates a CoreMessanger, bound to a single gate, so every
// forwarded call automatically carries the x-webitel-via header — mirroring
// whatsapp's decoratedCoreMessanger.
type viaCoreMessanger struct {
	CoreMessanger

	gateID string
}

func newViaCoreMessanger(messenger CoreMessanger, gateID uuid.UUID) *viaCoreMessanger {
	return &viaCoreMessanger{CoreMessanger: messenger, gateID: gateID.String()}
}

func (m *viaCoreMessanger) withVia(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, viaHeader, m.gateID)
}

func (m *viaCoreMessanger) SendText(ctx context.Context, in *coremodel.SendTextRequest) (*coremodel.SendTextResponse, error) {
	return m.CoreMessanger.SendText(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendImage(ctx context.Context, in *coremodel.SendImageRequest) (*coremodel.SendImageResponse, error) {
	return m.CoreMessanger.SendImage(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendDocument(ctx context.Context, in *coremodel.SendDocumentRequest) (*coremodel.SendDocumentResponse, error) {
	return m.CoreMessanger.SendDocument(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendLocation(ctx context.Context, in *coremodel.SendLocationRequest) (*coremodel.SendResponse, error) {
	return m.CoreMessanger.SendLocation(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendContact(ctx context.Context, in *coremodel.SendContactRequest) (*coremodel.SendResponse, error) {
	return m.CoreMessanger.SendContact(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendInteractiveCallback(ctx context.Context, in *coremodel.SendInteractiveCallbackRequest) error {
	return m.CoreMessanger.SendInteractiveCallback(m.withVia(ctx), in)
}

// coreMessengerFor returns a CoreMessanger bound to gate, automatically
// attaching the x-webitel-via header on every call.
func (p *Provider) coreMessengerFor(gate *model.Gate) CoreMessanger {
	return newViaCoreMessanger(p.coreMessageClient, gate.ID)
}
