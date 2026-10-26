package viberbm

import (
	"context"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	"github.com/webitel/im-providers-service/internal/core/via"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// viaCoreMessanger binds a Messenger to one gate, stamping the via headers.
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

func (p *viberBMProvider) coreMessengerFor(gate *vibbmmodel.ViberBMGate) sharedsvc.Messenger {
	return newViaCoreMessanger(p.messenger, via.Gate{ID: gate.ID, Type: p.Type(), Name: gate.Name})
}
