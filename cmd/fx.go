package cmd

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/health"
	healthfx "github.com/webitel/webitel-go-kit/infra/health/fx"
	healthhttp "github.com/webitel/webitel-go-kit/infra/health/http"

	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/infra/auth/standard"
	imauth "github.com/webitel/im-providers-service/infra/client/grpc/im-auth"
	imcontact "github.com/webitel/im-providers-service/infra/client/grpc/im-contact"
	imgateway "github.com/webitel/im-providers-service/infra/client/grpc/im-gateway"
	imthread "github.com/webitel/im-providers-service/infra/client/grpc/im-thread"
	"github.com/webitel/im-providers-service/infra/client/grpc/storage"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	httpsrv "github.com/webitel/im-providers-service/infra/srv/http"
	"github.com/webitel/im-providers-service/infra/tls"
	"github.com/webitel/im-providers-service/internal/core"
	sharedhandler "github.com/webitel/im-providers-service/internal/core/handler"
	"github.com/webitel/im-providers-service/internal/core/webhook"
	"github.com/webitel/im-providers-service/internal/custom"
	"github.com/webitel/im-providers-service/internal/facebook"
	"github.com/webitel/im-providers-service/internal/instagram"
	"github.com/webitel/im-providers-service/internal/provider"
	telegrambot "github.com/webitel/im-providers-service/internal/telegram/bot"
	"github.com/webitel/im-providers-service/internal/viber"
	"github.com/webitel/im-providers-service/internal/viberbm"
	"github.com/webitel/im-providers-service/internal/whatsapp"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

func NewApp(cfg *config.Config) *fx.App {
	return fx.New(AppOptions(cfg))
}

func AppOptions(cfg *config.Config) fx.Option {
	return fx.Options(
		fx.Supply(cfg),
		healthfx.Module(healthfx.Config{}),
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
		imthread.Module,
		storage.Module,
		imauth.Module,
		imcontact.Module,
		core.Module,
		facebook.Module,
		instagram.Module,
		whatsapp.Module,
		viber.Module,
		viberbm.Module,
		telegrambot.Module,
		custom.Module,
		webhook.Module,
		grpcsrv.Module,
		httpsrv.Module,
		sharedhandler.Module,

		fx.Invoke(registerHealth),
		healthfx.Shutdown(),
	)
}

// ProvideRouter sets up the Chi router with dynamic path parameters.
func ProvideRouter(wh *webhook.Handler, cfg *config.Config, logger *slog.Logger, h *health.Registry) http.Handler {
	r := chi.NewRouter()

	r.Handle("/livez", healthhttp.LivenessHandler(h, healthhttp.WithLogger(logger)))
	r.Handle("/readyz", healthhttp.ReadinessHandler(h, healthhttp.WithLogger(logger)))
	r.Handle("/healthz", healthhttp.HealthHandler(h, healthhttp.WithLogger(logger)))

	// Sanitize base path (e.g., "/wh")
	path := "/" + strings.Trim(cfg.Service.WebhookPath, "/")
	if path == "/" {
		path = "/wh"
	}

	// [DYNAMIC_PATTERN]: Adding /{uri} allows one route to handle infinite apps.
	// This will match: /wh/facebook/app-one, /wh/facebook/marketing-bot, etc.
	fullPath := path + "/{provider}/{uri}"

	logger.Info("registering dynamic webhook route",
		"pattern", fullPath,
	)

	// Handle both GET (verify) and POST (events)
	r.HandleFunc(fullPath, wh.ServeHTTP)

	return r
}
