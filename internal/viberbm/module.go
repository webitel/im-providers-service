package viberbm

import (
	"go.uber.org/fx"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	"github.com/webitel/im-providers-service/internal/provider"
	vibbmhandler "github.com/webitel/im-providers-service/internal/viberbm/handler"
	vibbmservice "github.com/webitel/im-providers-service/internal/viberbm/service"
	vibbmstore "github.com/webitel/im-providers-service/internal/viberbm/store"
	vibbmpostgres "github.com/webitel/im-providers-service/internal/viberbm/store/postgres"
)

var Module = fx.Module("viber_bm",
	fx.Provide(
		newAPIClient,

		// Adapter shared by the webhook registry (grouped provider.Provider) and
		// the service (TemplateSender).
		New,
		fx.Annotate(
			func(p *viberBMProvider) provider.Provider { return p },
			fx.ResultTags(`group:"providers"`),
		),
		func(p *viberBMProvider) vibbmservice.TemplateSender { return p },

		fx.Annotate(vibbmpostgres.NewViberBMStore, fx.As(new(vibbmstore.ViberBMStore))),

		fx.Annotate(vibbmservice.NewViberBMService, fx.As(new(vibbmservice.ViberBMManager))),

		vibbmhandler.NewViberBMHandler,
	),
	fx.Invoke(RegisterViberBMServices),
)

func RegisterViberBMServices(server *grpcsrv.Server, h *vibbmhandler.ViberBMHandler) {
	impb.RegisterViberBmServiceServer(server.Server, h)
}
