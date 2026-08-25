package instagram

import (
	"context"

	"google.golang.org/grpc/metadata"

	sharedmodel "github.com/webitel/im-providers-service/internal/core/model"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	igmodel "github.com/webitel/im-providers-service/internal/instagram/model"
)

// viaHeader identifies which gate a message forwarded to im-gateway-service
// belongs to, so the same external contact sub can be disambiguated across
// multiple Instagram Business Accounts. Mirrors facebook's and whatsapp's x-webitel-via.
const viaHeader = "x-webitel-via" //nolint:unused // will be used in viaCoreMessanger

// viaCoreMessanger decorates a sharedsvc.Messenger, bound to a single gate, so
// every forwarded call automatically carries the x-webitel-via header.
type viaCoreMessanger struct { //nolint:unused // will be used in NewViaCoreMessanger
	sharedsvc.Messenger

	gateID string
}

func newViaCoreMessanger(messenger sharedsvc.Messenger, gateID string) *viaCoreMessanger { //nolint:unused // called by coreMessengerFor
	return &viaCoreMessanger{Messenger: messenger, gateID: gateID}
}

func (m *viaCoreMessanger) withVia(ctx context.Context) context.Context { //nolint:unused // called by SendText et al
	return metadata.AppendToOutgoingContext(ctx, viaHeader, m.gateID)
}

func (m *viaCoreMessanger) SendText(ctx context.Context, in *sharedmodel.SendTextRequest) (*sharedmodel.SendTextResponse, error) { //nolint:unused // implements Messenger interface
	return m.Messenger.SendText(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendImage(ctx context.Context, in *sharedmodel.SendImageRequest) (*sharedmodel.SendImageResponse, error) { //nolint:unused // implements Messenger interface
	return m.Messenger.SendImage(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendDocument(ctx context.Context, in *sharedmodel.SendDocumentRequest) (*sharedmodel.SendDocumentResponse, error) { //nolint:unused // implements Messenger interface
	return m.Messenger.SendDocument(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendLocation(ctx context.Context, in *sharedmodel.SendLocationRequest) (*sharedmodel.SendResponse, error) { //nolint:unused // implements Messenger interface
	return m.Messenger.SendLocation(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendContact(ctx context.Context, in *sharedmodel.SendContactRequest) (*sharedmodel.SendResponse, error) { //nolint:unused // implements Messenger interface
	return m.Messenger.SendContact(m.withVia(ctx), in)
}

func (m *viaCoreMessanger) SendInteractiveCallback(ctx context.Context, in *sharedmodel.SendInteractiveCallbackRequest) error { //nolint:unused // implements Messenger interface
	return m.Messenger.SendInteractiveCallback(m.withVia(ctx), in)
}

// coreMessengerFor returns a Messenger bound to gate, automatically attaching
// the x-webitel-via header on every call.
func (p *instagramProvider) coreMessengerFor(gate *igmodel.InstagramGate) sharedsvc.Messenger { //nolint:unused // will be called in webhook handlers
	return newViaCoreMessanger(p.messenger, gate.ID)
}
