package viber

import (
	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	"github.com/webitel/im-providers-service/internal/provider"
	vibhandler "github.com/webitel/im-providers-service/internal/viber/handler"
	vibservice "github.com/webitel/im-providers-service/internal/viber/service"
	vibstore "github.com/webitel/im-providers-service/internal/viber/store"
	vibpostgres "github.com/webitel/im-providers-service/internal/viber/store/postgres"
	"go.uber.org/fx"
)

var Module = fx.Module("viber",
	fx.Provide(
		newClient,
		func(c *client) vibservice.ProviderAPI { return c },

		fx.Annotate(
			New,
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),

		fx.Annotate(vibpostgres.NewViberStore, fx.As(new(vibstore.ViberStore))),

		fx.Annotate(vibservice.NewViberService, fx.As(new(vibservice.ViberManager))),

		vibhandler.NewViberHandler,
	),
	fx.Invoke(RegisterViberServices),
)

func RegisterViberServices(server *grpcsrv.Server, viber *vibhandler.ViberHandler) {
	impb.RegisterViberServiceServer(server.Server, viber)
}
