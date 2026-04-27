package gate

import (
	"log/slog"

	"github.com/webitel/im-providers-service/infra/db/postgresx"
)

type gateModule struct {
	GateServer *whatsAppBusinessAccountServer
}

func NewGateModule(
	logger *slog.Logger,
	db postgresx.DB,
	internalContactResolver InternalContactResolver,
	encryptor Encryptor,
) *gateModule {
	gateRepository := newGateRepository(db)
	gateEditor := newGate(logger, gateRepository, internalContactResolver)
	gateGRPCServer := newWhatsAppBusinessAccountServer(gateEditor)

	return &gateModule{
		GateServer: gateGRPCServer,
	}
}
