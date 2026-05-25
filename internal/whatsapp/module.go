package whatsapp

import (
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	"github.com/webitel/im-providers-service/internal/provider"
	wahandler "github.com/webitel/im-providers-service/internal/whatsapp/handler"
	wasvc "github.com/webitel/im-providers-service/internal/whatsapp/service"
	wastore "github.com/webitel/im-providers-service/internal/whatsapp/store"
	"go.uber.org/fx"
)

// Module provides the WhatsApp adapter, store, service, and handler to the application.
var Module = fx.Module("whatsapp",
	fx.Provide(
		// Provider adapter
		fx.Annotate(
			New,
			fx.As(new(provider.Sender)),
			fx.As(new(provider.Receiver)),
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),

		// Store implementation
		fx.Annotate(wastore.NewWhatsAppStore, fx.As(new(wastore.WhatsAppStore))),

		// Service
		fx.Annotate(wasvc.NewWhatsAppService, fx.As(new(wasvc.WhatsAppManager))),

		// gRPC handler
		wahandler.NewWhatsAppHandler,
	),
	fx.Invoke(RegisterWhatsAppServices),
)

// RegisterWhatsAppServices connects the WhatsApp gRPC handler to the gRPC server.
func RegisterWhatsAppServices(
	server *grpcsrv.Server,
	whatsapp *wahandler.WhatsappHandler,
) {
	impb.RegisterWhatsAppServiceServer(server.Server, whatsapp)
}
