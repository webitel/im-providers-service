package instagram

import (
	"go.uber.org/fx"

	ighandler "github.com/webitel/im-providers-service/internal/instagram/handler"
	igservice "github.com/webitel/im-providers-service/internal/instagram/service"
	igstore "github.com/webitel/im-providers-service/internal/instagram/store"
	igpostgres "github.com/webitel/im-providers-service/internal/instagram/store/postgres"
	"github.com/webitel/im-providers-service/internal/provider"
)

var Module = fx.Module("instagram",
	fx.Provide(
		// Graph API client
		NewAPIClient,

		// Provider adapter
		fx.Annotate(
			New,
			fx.As(new(provider.Provider)),
			fx.ResultTags(`group:"providers"`),
		),

		// Store implementations
		fx.Annotate(igpostgres.NewInstagramStore, fx.As(new(igstore.InstagramStore))),

		// Services
		fx.Annotate(igservice.NewInstagramService, fx.As(new(igservice.InstagramManager))),
		fx.Annotate(igservice.NewInstagramOAuthService, fx.As(new(igservice.InstagramOAuthManager))),
	),
	ighandler.Module,
)
