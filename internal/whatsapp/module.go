package whatsapp

import (
	"log/slog"

	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/im-providers-service/internal/whatsapp/gate"
	"github.com/webitel/im-providers-service/pkg/crypto"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"whatsapp",
	fx.Provide(
		func(logger *slog.Logger, db postgresx.DB, internalContactResolver *imgateway.Client, encryptor crypto.Encryptor) WhatsAppGateServer {
			gateWire := gate.NewGateModule(logger, db, internalContactResolver, encryptor)
			return gateWire.GateServer
		},
	),
)
