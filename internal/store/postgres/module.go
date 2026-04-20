package postgres

import (
	"context"
	"log/slog"

	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/infra/db/pg"
	"go.uber.org/fx"
)

var Module = fx.Module("store",
	fx.Provide(
		ProvideNewDBConnection,
		pg.ProvidePgxPool,
		NewGateStore,
		NewMetaAppStore,
		NewFacebookStore,
		NewWhatsAppStore,

		// 2. Provide the Root Store aggregator as the main interface
		// This will satisfy dependencies on store.Store
		NewStore,
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
