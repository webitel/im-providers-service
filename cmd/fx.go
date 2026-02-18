package cmd

import (
	"github.com/go-chi/chi/v5"
	"github.com/webitel/im-providers-service/config"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	httpsrv "github.com/webitel/im-providers-service/infra/srv/http"
	"github.com/webitel/im-providers-service/internal/handler/http/webhook"
	"github.com/webitel/im-providers-service/internal/provider/facebook"
	"github.com/webitel/im-providers-service/internal/provider/whatsapp"
	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	"go.uber.org/fx"
)

// NewApp assembles the application dependency graph and manages its lifecycle.
func NewApp(cfg *config.Config) *fx.App {
	return fx.New(
		// [CORE] Essential application-wide dependencies.
		fx.Provide(
			func() *config.Config { return cfg },
			ProvideLogger,
			ProvideSD,
			ProvidePubSub,
		),

		// [INIT] Global service discovery orchestration.
		fx.Invoke(func(discovery discovery.DiscoveryProvider) error { return nil }),

		// [INFRA_CLIENTS] Internal gRPC clients for Webitel core services.
		// Manages connections to im-gateway, chat, and settings.
		imgateway.Module,

		// [SERVICES] Business logic layer.
		// Provides Messenger and Auther implementations.
		service.Module,

		// [ADAPTERS] External messenger platform implementations.
		// Each module registers itself into the "providers" value group.
		facebook.Module,
		whatsapp.Module,

		// [TRANSPORT] High-level protocol handlers and routing logic.
		webhook.Module,

		// [SERVERS] Network listeners (gRPC for internal API, HTTP for webhooks).
		grpcsrv.Module,
		httpsrv.Module,

		// [BOOTSTRAP] Orchestrate HTTP routing and application entry points.
		fx.Invoke(func(wh *webhook.Handler) {
			r := chi.NewRouter()

			// [WEBHOOKS] Centralized entry point for all external messenger events.
			// Pattern: POST /wh/facebook, /wh/whatsapp
			r.Post("/wh/{provider}", wh.ServeHTTP)

			// [TODO] Integrate the router 'r' with your httpsrv.Module
			// to start serving traffic on the configured port.
		}),
	)
}
