package grpc

import (
	"go.uber.org/fx"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
)

// Module registers all gRPC handlers and their registration logic.
var Module = fx.Module("provider-grpc",
	fx.Provide(
		NewGateHandler,
		NewFacebookHandler,
		NewMetaAppHandler,
		NewMetaOauthHandler,
		NewWhatsAppHandler,
	),
	fx.Invoke(RegisterProviderServices),
)

// RegisterProviderServices connects our internal handlers to the actual gRPC server.
func RegisterProviderServices(
	server *grpcsrv.Server,
	gate *GateHandler,
	facebook *FacebookHandler,
	metaApp *MetaAppHandler,
	metaOAuth *MetaOauthHandler,
	whatsapp *WhatsappHandler,
) {
	// Register each service defined in your proto files
	impb.RegisterGateServiceServer(server.Server, gate)
	impb.RegisterFacebookServiceServer(server.Server, facebook)
	impb.RegisterMetaAppServiceServer(server.Server, metaApp)
	impb.RegisterMetaOAuthServiceServer(server.Server, metaOAuth)
	impb.RegisterWhatsAppServiceServer(server.Server, whatsapp)
}
