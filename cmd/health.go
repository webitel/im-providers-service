package cmd

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/webitel/webitel-go-kit/infra/health"

	"github.com/webitel/im-providers-service/infra/db/postgresx"
	grpcsrv "github.com/webitel/im-providers-service/infra/srv/grpc"
)

func registerHealth(
	h *health.Registry,
	grpcServer *grpcsrv.Server,
	pool *pgxpool.Pool,
	db postgresx.DB,
	rdb *redis.Client,
) {
	h.Critical("grpc", health.ListenerCheck(grpcServer.Listener()))
	h.Informational("postgres", pool.Ping)
	h.Informational("postgres_whatsapp", db.Ping)
	h.Informational("redis", func(ctx context.Context) error { return rdb.Ping(ctx).Err() })
}
