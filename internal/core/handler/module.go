package handler

import (
	"go.uber.org/fx"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	"github.com/webitel/im-providers-service/internal/whatsapp"
)

// Module registers shared gRPC handlers and their registration logic.
var Module = fx.Module("provider-grpc",
	fx.Provide(
		NewGateHandler,
		NewOutboundMessageHandler,
	),
	fx.Invoke(RegisterSharedServices),
)

// RegisterSharedServices connects shared handlers to the gRPC server.
func RegisterSharedServices(
	server *grpcsrv.Server,
	gate *GateHandler,
	outboundMessage *OutboundMessageHandler,
	whatsAppServer whatsapp.WhatsAppGateServer,
) {
	impb.RegisterGateServiceServer(server.Server, gate)
	impb.RegisterProviderMessageServiceServer(server.Server, outboundMessage)
	impb.RegisterWhatsAppServiceServer(server.Server, whatsAppServer)
}
