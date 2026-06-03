package gate

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/gen/go/gateway/v1"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/im-providers-service/pkg/crypto"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type contactClientAdapter struct {
	client *imgateway.Client
}

func newContactClientAdapter(client *imgateway.Client) *contactClientAdapter {
	return &contactClientAdapter{client: client}
}

func (contactClient *contactClientAdapter) Search(ctx context.Context, peer Peer) (Peer, error) {
	onlyBots := true
	response, err := contactClient.client.Search(ctx, &gateway.SearchContactRequest{
		Size:     1,
		Subjects: []string{peer.GetSub()},
		OnlyBots: &onlyBots,
	})

	if err != nil {
		return nil, errors.Internal("executing gateway contact search request", errors.WithCause(err), errors.WithID("gate.wire.search"))
	}

	if len(response.GetItems()) == 0 {
		return nil, errors.InvalidArgument("zero records found for given bot identity", errors.WithID("gate.wire.search"))
	}

	return nil, nil
}

type gateModule struct {
	GateServer *whatsAppBusinessAccountServer
}

func NewGateModule(logger *slog.Logger, db postgresx.DB, client *imgateway.Client, encryptor crypto.Encryptor) *gateModule {
	var (
		gateRepository        = newGateRepository(db)
		internalContactClient = newContactClientAdapter(client)
		gateEditor            = newGate(logger, gateRepository, internalContactClient, encryptor)
		gateGRPCServer        = newWhatsAppBusinessAccountServer(gateEditor)
	)

	return &gateModule{
		GateServer: gateGRPCServer,
	}
}
