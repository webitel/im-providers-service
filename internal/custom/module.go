package custom

import (
	"go.uber.org/fx"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	customhandler "github.com/webitel/im-providers-service/internal/custom/handler"
	customservice "github.com/webitel/im-providers-service/internal/custom/service"
	customstore "github.com/webitel/im-providers-service/internal/custom/store"
	custompostgres "github.com/webitel/im-providers-service/internal/custom/store/postgres"
	"github.com/webitel/im-providers-service/internal/provider"
)

var Module = fx.Module("custom",
	fx.Provide(
		newClient,
		newFetcher,
		newRetryQueue,

		fx.Annotate(
			New,
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),

		fx.Annotate(custompostgres.NewCustomStore, fx.As(new(customstore.CustomStore))),

		fx.Annotate(customservice.NewCustomService, fx.As(new(customservice.CustomManager))),

		customhandler.NewCustomHandler,
	),
	fx.Invoke(RegisterCustomServices),
)

func RegisterCustomServices(server *grpcsrv.Server, custom *customhandler.CustomHandler) {
	impb.RegisterCustomServiceServer(server.Server, custom)
}
