package cmd

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/redis/go-redis/v9"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/internal/core/model"
	"github.com/webitel/webitel-go-kit/infra/discovery"
	_ "github.com/webitel/webitel-go-kit/infra/discovery/consul"
	otelsdk "github.com/webitel/webitel-go-kit/infra/otel/sdk"
	"github.com/webitel/webitel-go-kit/pkg/depenlog"
	"github.com/webitel/webitel-go-kit/pkg/logger"
	wsemconv "github.com/webitel/webitel-go-kit/pkg/semconv"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	"go.uber.org/fx"

	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/log/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/metric/stdout"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/otlp"
	_ "github.com/webitel/webitel-go-kit/infra/otel/sdk/trace/stdout"
)

func ProvideWatermillLogger(l *slog.Logger) watermill.LoggerAdapter {
	return watermill.NewSlogLogger(l)
}

// ProvideLogger is the central logger construction point. depenlog.New installs
// the logger process-wide (slog.SetDefault) and routes grpc-go's global logger
// through it (UseGRPC), then returns the kit handle. It provides both the
// logger.Logger (for depenlog adapters: fx, HTTP) and the *slog.Logger via
// slog.Default() (for existing consumers).
func ProvideLogger(cfg *config.Config, lc fx.Lifecycle) (logger.Logger, *slog.Logger, error) {
	logSettings := cfg.Log

	if !logSettings.Console && !logSettings.Otel && logSettings.File == "" {
		logSettings.Console = true
	}

	dcfg := depenlog.Config{
		Level:   logSettings.Level,
		JSON:    logSettings.JSON,
		File:    logSettings.File,
		Console: logSettings.Console,
	}

	// When OTel log export is active, route slog through the OTel bridge so the
	// LoggerProvider owns schema, severity, and trace/span correlation. The
	// console/file sink is intentionally bypassed — no dual-output fan-out.
	var opts []depenlog.Option
	if logSettings.Otel {
		service := resource.NewSchemaless(
			semconv.ServiceName(model.ServiceName),
			semconv.ServiceVersion(model.Version),
			semconv.ServiceInstanceID(discovery.GenerateInstanceID(model.ServiceName)),
			semconv.ServiceNamespace(model.ServiceNamespace),
		)

		// The bridge callback fires only when a log exporter is actually
		// configured; if it never fires we keep depenlog's console/file sink so
		// logs are not silently dropped.
		var bridgeEnabled bool
		shutdown, err := otelsdk.Configure(context.Background(), otelsdk.WithResource(service),
			otelsdk.WithLogBridge(
				func() {
					bridgeEnabled = true
				},
			),
		)
		if err != nil {
			return nil, nil, err
		}

		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return shutdown(ctx)
			},
		})

		if bridgeEnabled {
			opts = append(opts, depenlog.WithHandler(otelslog.NewHandler("slog")))
		}
	}

	kit := depenlog.New(dcfg, opts...)

	return kit, slog.Default(), nil
}

func ProvideSD(cfg *config.Config, log *slog.Logger, lc fx.Lifecycle) (discovery.DiscoveryProvider, error) {
	provider, err := discovery.DefaultFactory.CreateProvider(
		discovery.ProviderConsul,
		log,
		cfg.Consul.Addr,
		discovery.WithHeartbeat[discovery.DiscoveryProvider](true),
		discovery.WithTimeout[discovery.DiscoveryProvider](time.Second*30),
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
			err := rdb.Ping(ctx).Err()
			err = nil
			if err != nil {
				l.Error("redis connection failed", slog.Any(wsemconv.ErrorKey, err))
				return err
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
