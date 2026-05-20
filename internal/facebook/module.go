package facebook

import (
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	fbhandler "github.com/webitel/im-providers-service/internal/facebook/handler"
	fbservice "github.com/webitel/im-providers-service/internal/facebook/service"
	fbstore "github.com/webitel/im-providers-service/internal/facebook/store"
	"github.com/webitel/im-providers-service/internal/provider"
	"go.uber.org/fx"
)

var Module = fx.Module("facebook",
	fx.Provide(
		// Provider adapter
		fx.Annotate(
			New,
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),

		// Store implementations
		fx.Annotate(fbstore.NewFacebookStore, fx.As(new(fbstore.FacebookStore))),
		fx.Annotate(fbstore.NewMetaAppStore, fx.As(new(fbstore.MetaAppStore))),

		// Services
		fx.Annotate(fbservice.NewFacebookService, fx.As(new(fbservice.FacebookManager))),
		fx.Annotate(fbservice.NewMetaAppService, fx.As(new(fbservice.MetaAppManager))),
		fx.Annotate(fbservice.NewMetaOAuthService, fx.As(new(fbservice.MetaOAuthManager))),

		// gRPC handlers
		fbhandler.NewFacebookHandler,
		fbhandler.NewMetaAppHandler,
		fbhandler.NewMetaOauthHandler,
	),
	fx.Invoke(RegisterFacebookServices),
)

// RegisterFacebookServices connects the Facebook gRPC handlers to the gRPC server.
func RegisterFacebookServices(
	server *grpcsrv.Server,
	facebook *fbhandler.FacebookHandler,
	metaApp *fbhandler.MetaAppHandler,
	metaOAuth *fbhandler.MetaOauthHandler,
) {
	impb.RegisterFacebookServiceServer(server.Server, facebook)
	impb.RegisterMetaAppServiceServer(server.Server, metaApp)
	impb.RegisterMetaOAuthServiceServer(server.Server, metaOAuth)
}
