package viberbm

import (
	"context"

	"google.golang.org/grpc/metadata"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	vibbmmodel "github.com/webitel/im-providers-service/internal/viberbm/model"
)

// viaHeader disambiguates the same contact sub across gates on im-gateway.
// Mirrors facebook/whatsapp/telegram x-webitel-via.
const viaHeader = "x-webitel-via"

// viaCoreMessanger binds a Messenger to one gate, stamping x-webitel-via.
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

func (p *viberBMProvider) coreMessengerFor(gate *vibbmmodel.ViberBMGate) sharedsvc.Messenger {
	return newViaCoreMessanger(p.messenger, gate.ID)
}
