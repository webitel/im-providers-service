package httpsrv

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/webitel/im-providers-service/config"
	"go.uber.org/fx"
)

// Module registers the HTTP server lifecycle.
var Module = fx.Module("http-server",
	fx.Provide(New),
	fx.Invoke(func(*Server) {}),
)

// Server is the webhook HTTP server bound to its listener.
type Server struct {
	*http.Server

	listener net.Listener
}

// New binds cfg.Service.HTTPAddr and serves handler on it once the application starts.
func New(lc fx.Lifecycle, handler http.Handler, log *slog.Logger, cfg *config.Config) *Server {
	s := &Server{Server: &http.Server{
		Addr:    cfg.Service.HTTPAddr,
		Handler: handler,
	}}
	srv := s.Server

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var lcfg net.ListenConfig

			l, err := lcfg.Listen(ctx, "tcp", srv.Addr)
			if err != nil {
				return fmt.Errorf("listen http %s: %w", srv.Addr, err)
			}

			s.listener = l

			log.Info("HTTP_SERVER_STARTED", slog.String("addr", srv.Addr))
			go func() {
				if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
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

	return s
}

// Listener returns the bound listener; nil until the application starts.
func (s *Server) Listener() net.Listener {
	return s.listener
}
