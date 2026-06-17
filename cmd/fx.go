package cmd

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/infra/auth/standard"
	imauth "github.com/webitel/im-providers-service/infra/client/grpc/im-auth"
	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	"github.com/webitel/im-providers-service/infra/client/grpc/storage"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	httpsrv "github.com/webitel/im-providers-service/infra/srv/http"
	"github.com/webitel/im-providers-service/infra/tls"
	"github.com/webitel/im-providers-service/internal/core"
	sharedhandler "github.com/webitel/im-providers-service/internal/core/handler"
	"github.com/webitel/im-providers-service/internal/core/webhook"
	"github.com/webitel/im-providers-service/internal/facebook"
	"github.com/webitel/im-providers-service/internal/provider"
	"github.com/webitel/im-providers-service/internal/whatsapp"
	"github.com/webitel/im-providers-service/pkg/crypto"
	"github.com/webitel/webitel-go-kit/pkg/depenlog"
	"github.com/webitel/webitel-go-kit/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func NewApp(cfg *config.Config) *fx.App {
	return fx.New(
		fx.Supply(cfg),
		fx.WithLogger(func(l logger.Logger) fxevent.Logger {
			return depenlog.FxLogger(l)
		}),
		fx.Provide(
			ProvideLogger,
			ProvideWatermillLogger,
			ProvideSD,
			ProvideRouter,
			ProvideRedis,
		),
		provider.Module,
		standard.Module,
		tls.Module,
		crypto.Module,
		imgateway.Module,
		storage.Module,
		imauth.Module,
		imcontact.Module,
		core.Module,
		facebook.Module,
		whatsapp.Module,
		webhook.Module,
		grpcsrv.Module,
		httpsrv.Module,
		sharedhandler.Module,
	)
}

// ProvideRouter sets up the Chi router with dynamic path parameters.
// depenlog.Middleware logs every request through the unified logger using the
// request context, and otelhttp wraps the router so each request carries an
// active span — so those request logs include trace_id/span_id.
func ProvideRouter(wh *webhook.Handler, cfg *config.Config, log logger.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(depenlog.Middleware(log))

	// Sanitize base path (e.g., "/wh")
	path := "/" + strings.Trim(cfg.Service.WebhookPath, "/")
	if path == "/" {
		path = "/wh"
	}

	// [DYNAMIC_PATTERN]: Adding /{uri} allows one route to handle infinite apps.
	// This will match: /wh/facebook/app-one, /wh/facebook/marketing-bot, etc.
	fullPath := path + "/{provider}/{uri}"

	log.Info("registering dynamic webhook route",
		"pattern", fullPath,
	)

	// Handle both GET (verify) and POST (events)
	r.HandleFunc(fullPath, wh.ServeHTTP)

	return otelhttp.NewHandler(r, "webhook")
}
