package facebook

import (
	"context"

	"google.golang.org/grpc/metadata"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
)

// viaHeader identifies which gate a message forwarded to im-gateway-service
// belongs to, so the same external contact sub can be disambiguated across
// multiple Facebook pages. Mirrors whatsapp's and telegram's x-webitel-via.
const viaHeader = "x-webitel-via"

// viaCoreMessanger decorates a sharedsvc.Messenger, bound to a single gate, so
// every forwarded call automatically carries the x-webitel-via header.
type viaCoreMessanger struct {
	sharedsvc.Messenger

	gateID string
}

func newViaCoreMessanger(messenger sharedsvc.Messenger, gateID string) *viaCoreMessanger {
	return &viaCoreMessanger{Messenger: messenger, gateID: gateID}
}

func (m *viaCoreMessanger) withVia(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, viaHeader, m.gateID)
}

func (m *viaCoreMessanger) SendText(ctx context.Context, in *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error) {
	return m.Messenger.SendText(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendImage(ctx context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) {
	return m.Messenger.SendImage(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendDocument(ctx context.Context, in *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error) {
	return m.Messenger.SendDocument(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendLocation(ctx context.Context, in *sharedmodel.SendLocationRequest) (*sharedmodel.SendResponse, error) {
	return m.Messenger.SendLocation(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendContact(ctx context.Context, in *sharedmodel.SendContactRequest) (*sharedmodel.SendResponse, error) {
	return m.Messenger.SendContact(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendInteractiveCallback(ctx context.Context, in *sharedmodel.SendInteractiveCallbackRequest) error {
	return m.Messenger.SendInteractiveCallback(m.withVia(ctx), in)
}

// coreMessengerFor returns a Messenger bound to gate, automatically attaching
// the x-webitel-via header on every call.
func (p *facebookProvider) coreMessengerFor(gate *fbmodel.FacebookGate) sharedsvc.Messenger {
	return newViaCoreMessanger(p.messenger, gate.ID)
}
