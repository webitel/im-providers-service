package custom

import (
	"go.uber.org/fx"

	impb "github.com/webitel/im-providers-service/gen/go/provider/v1"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
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

		custompostgres.NewCustomStore,
		fx.Annotate(newCustomStore, fx.As(new(customstore.CustomStore))),
		newOutboxStore,

		fx.Annotate(customservice.NewCustomService, fx.As(new(customservice.CustomManager))),

		customhandler.NewCustomHandler,
	),
	fx.Invoke(RegisterCustomServices),
)

func newCustomStore(base *custompostgres.CustomStore, cache sharedstore.GateCache) *customstore.CachedStore {
	return customstore.NewCachedStore(base, cache)
}

func newOutboxStore(base *custompostgres.CustomStore) customstore.OutboxStore {
	return base
}

func RegisterCustomServices(server *grpcsrv.Server, custom *customhandler.CustomHandler) {
	impb.RegisterCustomServiceServer(server.Server, custom)
}
