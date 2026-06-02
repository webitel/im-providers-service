package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/webitel-go-kit/appconfig"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type Config struct {
	Service  ServiceConfig    `mapstructure:"service"`
	Log      appconfig.Log    `mapstructure:"log"`
	Postgres PostgresConfig   `mapstructure:"postgres"`
	Redis    appconfig.Redis  `mapstructure:"redis"`
	Consul   appconfig.Consul `mapstructure:"consul"`
}

type ServiceConfig struct {
	ID          string             `mapstructure:"id"`
	GRPCAddr    string             `mapstructure:"addr"`
	HTTPAddr    string             `mapstructure:"http_addr"`
	WebhookPath string             `mapstructure:"webhook_path"`
	Connection  appconfig.GRPCConn `mapstructure:"conn"`
	SecretKey   string             `mapstructure:"secret_key"`
}

// PostgresConfig extends the basic DSN with connection-pool options specific to this service.
type PostgresConfig struct {
	DSN      string   `mapstructure:"dsn"`
	Replicas []string `mapstructure:"replicas"`

	QueryTimeout time.Duration `mapstructure:"query_timeout"`
	TxTimeout    time.Duration `mapstructure:"tx_timeout"`

	MaxConns          int32         `mapstructure:"max_conns"`
	MinConns          int32         `mapstructure:"min_conns"`
	MaxConnLifetime   time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `mapstructure:"health_check_period"`
	ConnectTimeout    time.Duration `mapstructure:"connect_timeout"`

	ApplicationName     string `mapstructure:"application_name"`
	LoadBalancingPolicy string `mapstructure:"lb_policy"`
}

func (p *PostgresConfig) ToOpenOptions() []postgresx.OpenOption {
	var opts []postgresx.OpenOption
	if len(p.Replicas) > 0 {
		opts = append(opts, postgresx.WithReplicas(p.Replicas...))
	}
	if p.QueryTimeout > 0 {
		opts = append(opts, postgresx.WithQueryTimeout(p.QueryTimeout))
	}
	if p.TxTimeout > 0 {
		opts = append(opts, postgresx.WithTxTimeout(p.TxTimeout))
	}

	var poolOpts []postgresx.PoolOption
	if p.MaxConns > 0 {
		poolOpts = append(poolOpts, postgresx.WithMaxConns(p.MaxConns))
	}
	if p.MinConns > 0 {
		poolOpts = append(poolOpts, postgresx.WithMinConns(p.MinConns))
	}
	if p.MaxConnLifetime > 0 {
		poolOpts = append(poolOpts, postgresx.WithMaxConnLifetime(p.MaxConnLifetime))
	}
	if p.MaxConnIdleTime > 0 {
		poolOpts = append(poolOpts, postgresx.WithMaxConnIdleTime(p.MaxConnIdleTime))
	}
	if p.ConnectTimeout > 0 {
		poolOpts = append(poolOpts, postgresx.WithConnectTimeout(p.ConnectTimeout))
	}
	if p.ApplicationName != "" {
		poolOpts = append(poolOpts, postgresx.WithApplicationName(p.ApplicationName))
	}

	if len(poolOpts) > 0 {
		opts = append(opts, postgresx.WithPoolOptions(poolOpts...))
	}

	return opts
}

// loader is the single server command's config loader.
// Postgres flags are registered manually below because this service extends
// the base DSN with pool options not covered by appconfig.Sections.Postgres.
var loader = appconfig.NewLoader(appconfig.Sections{
	Log:    true,
	Redis:  true,
	Consul: true,
})

// LoadConfig reads configuration from flags, environment variables, and an
// optional YAML/JSON file. It also sets up hot-reload if a config file is used.
func LoadConfig() (*Config, error) {
	loader.RegisterFlags(pflag.CommandLine)
	registerServiceFlags()
	registerPostgresFlags()
	pflag.Parse()

	cfg := &Config{}
	if err := loader.Load(pflag.CommandLine, cfg); err != nil {
		return nil, err
	}

	loader.Watch(func(e fsnotify.Event) {
		slog.Info("config file changed", "name", e.Name)

		newCfg := &Config{}
		if err := loader.Viper().Unmarshal(newCfg); err != nil {
			slog.Error("config reload: unmarshal failed", "error", err)
			return
		}
		if err := newCfg.validate(); err != nil {
			slog.Error("config reload: validation failed", "error", err)
			return
		}
		*cfg = *newCfg
		slog.Info("config reloaded")
	})

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func registerServiceFlags() {
	pflag.String("service.id", "", "Service instance ID (required)")
	pflag.String("service.addr", "localhost:8080", "gRPC listen address")
	pflag.String("service.http_addr", ":8085", "HTTP listen address")
	pflag.String("service.webhook_path", "/wh", "Base path for incoming webhooks")
	pflag.String("service.secret_key", "", "32-byte AES key for token encryption (required)")

	pflag.Bool("service.conn.verify_certs", false, "Verify TLS certificates on outbound gRPC connections")
	pflag.String("service.conn.ca", "", "CA certificate path")
	pflag.String("service.conn.cert", "", "Server certificate path")
	pflag.String("service.conn.key", "", "Server certificate key path")
	pflag.String("service.conn.client.ca", "", "Client CA certificate path")
	pflag.String("service.conn.client.cert", "", "Client certificate path")
	pflag.String("service.conn.client.key", "", "Client certificate key path")
}

func registerPostgresFlags() {
	pflag.String("postgres.dsn", "", "PostgreSQL primary DSN (required)")
	pflag.StringSlice("postgres.replicas", []string{}, "Replica DSNs (comma-separated)")
	pflag.Duration("postgres.query_timeout", 0, "Default query timeout (0 = no timeout)")
	pflag.Duration("postgres.tx_timeout", 0, "Default transaction timeout (0 = no timeout)")
	pflag.Int32("postgres.max_conns", 0, "Max pool connections (0 = default)")
	pflag.Int32("postgres.min_conns", 0, "Min pool connections (0 = default)")
	pflag.Duration("postgres.max_conn_lifetime", 0, "Max connection lifetime (0 = no limit)")
	pflag.Duration("postgres.max_conn_idle_time", 0, "Max idle connection time (0 = no limit)")
	pflag.Duration("postgres.health_check_period", 0, "Pool health-check period (0 = default)")
	pflag.Duration("postgres.connect_timeout", 0, "Connection establishment timeout (0 = no timeout)")
	pflag.String("postgres.application_name", "webitel-im-provider", "application_name sent to PostgreSQL")
}

func (c *Config) validate() error {
	if c.Service.ID == "" {
		return fmt.Errorf("config: service.id is required")
	}
	if c.Service.GRPCAddr == "" {
		return fmt.Errorf("config: service.addr is required")
	}

	if c.Service.WebhookPath == "" {
		c.Service.WebhookPath = "/wh"
	}
	if !strings.HasPrefix(c.Service.WebhookPath, "/") {
		c.Service.WebhookPath = "/" + c.Service.WebhookPath
	}

	if err := appconfig.ValidateGRPCConn("service.conn", c.Service.Connection); err != nil {
		return err
	}

	if c.Postgres.DSN == "" {
		return errors.InvalidArgument("postgres.dsn is required", errors.WithID("config.config.validate"))
	}

	return nil
}
