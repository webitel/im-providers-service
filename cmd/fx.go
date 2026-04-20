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
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	httpsrv "github.com/webitel/im-providers-service/infra/srv/http"
	"github.com/webitel/im-providers-service/infra/tls"
	"github.com/webitel/im-providers-service/internal/handler/grpc"
	"github.com/webitel/im-providers-service/internal/handler/http/webhook"
	"github.com/webitel/im-providers-service/internal/provider/facebook"
	"github.com/webitel/im-providers-service/internal/provider/whatsapp"
	"github.com/webitel/im-providers-service/internal/service"
	"github.com/webitel/im-providers-service/internal/store/postgres"
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
		),

		standard.Module,
		tls.Module,
		crypto.Module,
		imgateway.Module,
		imauth.Module,
		imcontact.Module,
		service.Module,
		facebook.Module,
		whatsapp.Module,
		webhook.Module,
		grpcsrv.Module,
		httpsrv.Module,
		grpc.Module,
		postgres.Module,
	)
}

// ProvideRouter sets up the Chi router and returns it as an http.Handler.
func ProvideRouter(wh *webhook.Handler, cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	// Resolve and sanitize webhook path
	path := "/" + strings.Trim(cfg.Service.WebhookPath, "/")
	if path == "/" {
		path = "/wh"
	}

	// [WEBHOOKS] Centralized entry point
	r.Post(path+"/{provider}", wh.ServeHTTP)

	return r
}
