package storedi

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/infra/db/pg"
	"github.com/webitel/im-providers-service/infra/db/postgresx"

	"github.com/webitel/im-providers-service/internal/store"
	"github.com/webitel/im-providers-service/internal/store/lru"
	"github.com/webitel/im-providers-service/internal/store/postgres"
	redisstore "github.com/webitel/im-providers-service/internal/store/redis"
	"go.uber.org/fx"
)

var Module = fx.Module("store",
	fx.Provide(
		ProvideNewDBConnection,
		ProvideNewPostgresxConnection,
		pg.ProvidePgxPool,

		// Provide LRU Cache (size 1000 items)
		func() (store.GateCache, error) {
			return lru.NewLRUCache(1000)
		},

		func(rdb *redis.Client) store.ExternalUserCache {
			return nil
			// Identity TTL set to 24 hours
			return redisstore.NewRedisUserCache(rdb, 24*time.Hour)
		},

		postgres.NewGateStore,
		postgres.NewMetaAppStore,
		postgres.NewFacebookStore,
		postgres.NewWhatsAppStore,
		postgres.NewStore,
	),
)

func ProvideNewDBConnection(cfg *config.Config, l *slog.Logger, lc fx.Lifecycle) (*pg.PgxDB, error) {
	db, err := pg.New(context.Background(), l, cfg.Postgres.DSN)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			db.Master().Close()
			return nil
		},
	})

	return db, err
}

func ProvideNewPostgresxConnection(cfg *config.Config, l *slog.Logger, lc fx.Lifecycle) (postgresx.DB, error) {
	ctx := context.Background()
	db, err := postgresx.New(ctx, cfg.Postgres.DSN, cfg.Postgres.ToOpenOptions()...)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			db.Close()
			return nil
		},
	})

	return db, err
}
