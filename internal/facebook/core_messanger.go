package facebook

import (
	"context"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	"github.com/webitel/im-providers-service/internal/core/via"
	fbmodel "github.com/webitel/im-providers-service/internal/facebook/model"
)

// viaCoreMessanger decorates a sharedsvc.Messenger, bound to a single gate, so
// every forwarded call automatically carries the via headers (see package via).
type viaCoreMessanger struct {
	sharedsvc.Messenger

	gate via.Gate
}

func newViaCoreMessanger(messenger sharedsvc.Messenger, gate via.Gate) *viaCoreMessanger {
	return &viaCoreMessanger{Messenger: messenger, gate: gate}
}

func (m *viaCoreMessanger) withVia(ctx context.Context) context.Context {
	return via.AppendToOutgoingContext(ctx, m.gate)
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
// the via headers on every call.
func (p *facebookProvider) coreMessengerFor(gate *fbmodel.FacebookGate) sharedsvc.Messenger {
	return newViaCoreMessanger(p.messenger, via.Gate{ID: gate.ID, Type: p.Type(), Name: gate.Name})
}
