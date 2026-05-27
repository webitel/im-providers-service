package messaging

import (
	"log/slog"

	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/im-providers-service/internal/whatsapp/common"
)

type messagingWire struct {
	Messaging *Messaging
}

func NewMessagingWire(
	logger *slog.Logger,
	encryptor common.Encryptor,
	gatewayClient *imgateway.Client,
	db postgresx.DB,
) *messagingWire {
	messagingRepo := newMessagingRepository(db)

	return &messagingWire{
		Messaging: newMessaging(logger, encryptor, messagingRepo),
	}
}
