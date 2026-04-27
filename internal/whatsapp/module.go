package whatsapp

import (
	"log/slog"

	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/im-providers-service/internal/whatsapp/gate"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"whatsapp",
	fx.Provide(
		func(logger *slog.Logger, db postgresx.DB, internalContactResolver gate.InternalContactResolver, encryptor gate.Encryptor) WhatsAppGateServer {
			gateWire := gate.NewGateModule(logger, db, internalContactResolver, encryptor)
			return gateWire.GateServer
		},
	),
)
