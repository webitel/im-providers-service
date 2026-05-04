package cmd

import (
	"log/slog"
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
	"github.com/webitel/im-providers-service/internal/handler/grpc"
	"github.com/webitel/im-providers-service/internal/handler/http/webhook"
	"github.com/webitel/im-providers-service/internal/media"
	"github.com/webitel/im-providers-service/internal/provider/facebook"
	"github.com/webitel/im-providers-service/internal/service"
	storedi "github.com/webitel/im-providers-service/internal/store/di"
	"github.com/webitel/im-providers-service/internal/whatsapp"
	"github.com/webitel/im-providers-service/pkg/crypto"
	"go.uber.org/fx"
)

func NewApp(cfg *config.Config) *fx.App {
	return fx.New(
		fx.Supply(cfg),
		fx.Provide(
			ProvideLogger,
			ProvideWatermillLogger,
			ProvideSD,
			ProvideRouter,
			ProvideRedis,
		),

		standard.Module,
		tls.Module,
		crypto.Module,
		imgateway.Module,
		storage.Module,
		media.Module,
		imauth.Module,
		imcontact.Module,
		service.Module,
		facebook.Module,
		whatsapp.Module,
		webhook.Module,
		grpcsrv.Module,
		httpsrv.Module,
		grpc.Module,
		storedi.Module,
	)
}

// ProvideRouter sets up the Chi router with dynamic path parameters.
func ProvideRouter(wh *webhook.Handler, cfg *config.Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

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
