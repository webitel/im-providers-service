package cmd

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/webitel/webitel-go-kit/infra/discovery"
	"github.com/webitel/webitel-go-kit/infra/discovery/consul"
	"github.com/webitel/webitel-go-kit/infra/health"

	"github.com/webitel/im-providers-service/cmd/logging"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/internal/core/model"

	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/otlp" // register otel sdk drivers
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/stdout"
)

func ProvideWatermillLogger(l *slog.Logger) watermill.LoggerAdapter {
	return watermill.NewSlogLogger(l)
}

func ProvideLogger(cfg *config.Config, lc fx.Lifecycle) (*slog.Logger, error) {
	return logging.ProvideLogger(cfg, lc)
}

func ProvideSD(cfg *config.Config, log *slog.Logger, h *health.Registry, lc fx.Lifecycle) (discovery.DiscoveryProvider, error) {
	provider, err := discovery.DefaultFactory.CreateProvider(
		discovery.ProviderConsul,
		log,
		cfg.Consul.Addr,
		discovery.WithHeartbeat[discovery.DiscoveryProvider](true),
		discovery.WithTimeout[discovery.DiscoveryProvider](time.Second*30),
		consul.WithReadiness(h.ReadyFunc()),
	)
	if err != nil {
		return nil, err
	}

	si := new(discovery.ServiceInstance)
	{
		si.Id = discovery.GenerateInstanceID(model.ServiceName)
		si.Name = model.ServiceName
		si.Version = model.Version
		si.Metadata = map[string]string{
			"commit":         model.Commit,
			"commitDate":     model.CommitDate,
			"branch":         model.Branch,
			"buildTimestamp": model.BuildTimestamp,
		}
		si.Endpoints = []string{(&url.URL{Scheme: "grpc", Host: cfg.Service.GRPCAddr}).String()}
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := provider.Register(ctx, si); err != nil {
				return err
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := provider.Deregister(ctx, si); err != nil {
				return err
			}
			return nil
		},
	})

	return provider, nil
}

func ProvideRedis(cfg *config.Config, lc fx.Lifecycle, l *slog.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := rdb.Ping(ctx).Err(); err != nil {
				l.Warn("redis unreachable at startup", slog.String("addr", cfg.Redis.Addr), slog.Any("error", err))

				return nil
			}

			l.Info("redis connected", slog.String("addr", cfg.Redis.Addr))

			return nil
		},
		OnStop: func(ctx context.Context) error {
			l.Info("closing redis connection")
			return rdb.Close()
		},
	})

	return rdb, nil
}
