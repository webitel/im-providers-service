package migrate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/urfave/cli/v2"
	"go.uber.org/fx"

	"github.com/webitel/im-providers-service/cmd/logging"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/migrations"
)

func CMD() *cli.Command {
	return &cli.Command{
		Name:    "migrate",
		Aliases: []string{"m"},
		Usage:   "Execute database migrations",
		Action: func(c *cli.Context) error {
			cfg, err := config.LoadMigrateConfig()
			if err != nil {
				return err
			}

			app := fx.New(
				fx.Supply(cfg),
				fx.Provide(logging.ProvideLogger),
				fx.Invoke(func(cfg *config.Config, log *slog.Logger, _ fx.Lifecycle) error {
					return Run(c.Context, cfg, log)
				}),
				fx.NopLogger,
			)

			return app.Start(c.Context)
		},
	}
}

func Run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	conf, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return err
	}

	db := stdlib.OpenDB(*conf.ConnConfig)
	defer db.Close()

	goose.SetLogger(newLogger(log))
	goose.SetVerbose(true)

	store, err := database.NewStore(database.DialectPostgres, "im_providers_schema_version")
	if err != nil {
		return err
	}

	noopDialect := goose.Dialect("")

	provider, err := goose.NewProvider(noopDialect, db, migrations.EmbedMigrations, goose.WithStore(store))
	if err != nil {
		return err
	}

	res, err := provider.Up(ctx)
	if err != nil {
		return err
	}

	for _, r := range res {
		if r.Error != nil {
			log.Error("unable to apply migration", "err", r.Error)
		} else {
			log.Info("applied migration")
		}
	}

	return nil
}

type migrateLogger struct {
	log *slog.Logger
}

func newLogger(log *slog.Logger) *migrateLogger {
	return &migrateLogger{log: log}
}

func (l *migrateLogger) Printf(format string, args ...any) {
	l.log.Info(fmt.Sprintf(format, args...))
}

func (l *migrateLogger) Fatalf(format string, args ...any) {
	l.log.Error(fmt.Sprintf(format, args...))
}
