package httpsrv

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/webitel/im-providers-service/config"
	"go.uber.org/fx"
)

// Module registers the HTTP server lifecycle.
var Module = fx.Module("http-server",
	fx.Invoke(Start), // [LIFECYCLE] Starts the listener using the provided handler
)

// Start initializes the http.Server with a handler provided by the dependency graph.
func Start(lc fx.Lifecycle, handler http.Handler, log *slog.Logger, cfg *config.Config) {
	srv := &http.Server{
		Addr:    cfg.Service.HTTPAddr,
		Handler: handler,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("HTTP_SERVER_STARTED", slog.String("addr", srv.Addr))
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error("HTTP_SERVER_CRASHED", slog.Any("err", err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info("HTTP_SERVER_STOPPING")
			return srv.Shutdown(ctx)
		},
	})
}
