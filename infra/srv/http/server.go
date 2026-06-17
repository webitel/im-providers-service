package httpsrv

import (
	"context"
	"net/http"

	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/webitel-go-kit/pkg/depenlog"
	"github.com/webitel/webitel-go-kit/pkg/logger"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.uber.org/fx"
)

// Module registers the HTTP server lifecycle.
var Module = fx.Module("http-server",
	fx.Invoke(Start), // [LIFECYCLE] Starts the listener using the provided handler
)

// Start initializes the http.Server with a handler provided by the dependency
// graph. net/http's internal errors are routed into the unified logger via
// depenlog.ErrorLog.
func Start(lc fx.Lifecycle, handler http.Handler, log logger.Logger, cfg *config.Config) {
	srv := &http.Server{
		Addr:     cfg.Service.HTTPAddr,
		Handler:  handler,
		ErrorLog: depenlog.ErrorLog(log),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info("HTTP_SERVER_STARTED", "addr", srv.Addr)
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Error("HTTP_SERVER_CRASHED", semconv.ErrorKey, err)
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
