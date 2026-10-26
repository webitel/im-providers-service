package bot

import (
	"context"

	coremodel "github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/im-providers-service/internal/core/via"
	"github.com/webitel/im-providers-service/internal/telegram/bot/model"
)

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
// forwarded call automatically carries the via headers (see package via) — mirroring
// whatsapp's decoratedCoreMessanger.
type viaCoreMessanger struct {
	CoreMessanger

	gate via.Gate
}

func newViaCoreMessanger(messenger CoreMessanger, gate via.Gate) *viaCoreMessanger {
	return &viaCoreMessanger{CoreMessanger: messenger, gate: gate}
}

func (m *viaCoreMessanger) withVia(ctx context.Context) context.Context {
	return via.AppendToOutgoingContext(ctx, m.gate)
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
// attaching the via headers on every call.
func (p *Provider) coreMessengerFor(gate *model.Gate) CoreMessanger {
	return newViaCoreMessanger(p.coreMessageClient, via.Gate{ID: gate.ID.String(), Type: p.Type(), Name: gate.Name})
}
