package core

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/infra/db/pg"
	sharedsvc "github.com/webitel/im-providers-service/internal/core/service"
	sharedstore "github.com/webitel/im-providers-service/internal/core/store"
	"go.uber.org/fx"
)

var Module = fx.Module("shared",
	fx.Provide(
		ProvideNewDBConnection,

		pg.ProvidePgxPool,

		// Provide LRU Cache (size 1000 items)
		func() (sharedstore.GateCache, error) {
			return sharedstore.NewLRUCache(1000)
		},

		func(rdb *redis.Client) sharedstore.ExternalUserCache {
			// Identity TTL set to 24 hours
			return sharedstore.NewRedisUserCache(rdb, 24*time.Hour)
		},

		fx.Annotate(sharedstore.NewGateStore, fx.As(new(sharedstore.GateStore))),

		sharedsvc.NewMediaService,
		fx.Annotate(sharedsvc.NewGateService, fx.As(new(sharedsvc.GateManager))),
		fx.Annotate(sharedsvc.NewAuthService, fx.As(new(sharedsvc.Auther))),
		fx.Annotate(sharedsvc.NewMediaService, fx.As(new(sharedsvc.MediaManager))),

		fx.Annotate(
			sharedsvc.NewMessageService,
			fx.ResultTags(`name:"baseMessenger"`),
		),
		fx.Annotate(
			ProvideDecoratedMessenger,
			fx.ParamTags(`name:"baseMessenger"`),
			fx.As(new(sharedsvc.Messenger)),
		),
	),
)

func ProvideDecoratedMessenger(baseMessenger sharedsvc.Messenger) sharedsvc.Messenger {
	return sharedsvc.NewMessengerAuthMiddleware(baseMessenger)
}

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
