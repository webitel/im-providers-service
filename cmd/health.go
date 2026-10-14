package cmd

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/health"

	"github.com/webitel/im-providers-service/infra/db/postgresx"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
	httpsrv "github.com/webitel/im-providers-service/infra/srv/http"
)

func registerHealth(
	lc fx.Lifecycle,
	h *health.Registry,
	grpcServer *grpcsrv.Server,
	httpServer *httpsrv.Server,
	pool *pgxpool.Pool,
	db postgresx.DB,
	rdb *redis.Client,
) {
	lc.Append(fx.StartHook(func() {
		h.Critical("grpc", health.ListenerCheck(grpcServer.Listener()))
		h.Critical("http", health.ListenerCheck(httpServer.Listener()))
		h.Informational("postgres", pool.Ping)
		h.Informational("postgres_whatsapp", db.Ping)
		h.Informational("redis", func(ctx context.Context) error { return rdb.Ping(ctx).Err() })
	}))
}
