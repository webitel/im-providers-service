package httpsrv

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/webitel/im-providers-service/config"
	"go.uber.org/fx"
)

var Module = fx.Module("http-server",
	fx.Provide(http.NewServeMux), // [ROUTER] Provides central *http.ServeMux
	fx.Invoke(Start),             // [LIFECYCLE] Starts the listener
)

func Start(lc fx.Lifecycle, mux *http.ServeMux, log *slog.Logger, cfg *config.Config) {
	srv := &http.Server{
		Addr:    cfg.Service.HTTPAddr,
		Handler: mux,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("HTTP_SERVER_STARTED", slog.String("addr", srv.Addr))
			// [IO] Run in background to not block app startup
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
