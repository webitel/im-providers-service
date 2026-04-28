package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/webitel/im-providers-service/infra/db/postgresx"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

type Config struct {
	Service  ServiceConfig  `mapstructure:"service"`
	Log      LogConfig      `mapstructure:"log"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Consul   ConsulConfig   `mapstructure:"consul"`
}

type ServiceConfig struct {
	ID          string           `mapstructure:"id"`
	GRPCAddr    string           `mapstructure:"addr"`         // gRPC server address
	HTTPAddr    string           `mapstructure:"http_addr"`    // HTTP server address
	WebhookPath string           `mapstructure:"webhook_path"` // Base path for webhooks (e.g., /wh)
	Connection  ConnectionConfig `mapstructure:"conn"`
	SecretKey   string           `mapstructure:"secret_key"`
}

type ConnectionConfig struct {
	TLS         TLSConfig `mapstructure:",squash"`
	VerifyCerts bool      `mapstructure:"verify_certs"`
	Client      TLSConfig `mapstructure:"client"`
}

type TLSConfig struct {
	CA   string `mapstructure:"ca"`
	Cert string `mapstructure:"cert"`
	Key  string `mapstructure:"key"`
}

type LogConfig struct {
	Level   string `mapstructure:"level"`
	JSON    bool   `mapstructure:"json"`
	Otel    bool   `mapstructure:"otel"`
	File    string `mapstructure:"file"`
	Console bool   `mapstructure:"console"`
}

type PostgresConfig struct {
	DSN          string        `mapstructure:"dsn"`
	Replicas     []string      `mapstructure:"replicas"`
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

func (postgresConfig *PostgresConfig) ToOpenOptions() []postgresx.OpenOption {
	var opts []postgresx.OpenOption
	if len(postgresConfig.Replicas) > 0 {
		opts = append(opts, postgresx.WithReplicas(postgresConfig.Replicas...))
	}
	if postgresConfig.QueryTimeout > 0 {
		opts = append(opts, postgresx.WithQueryTimeout(postgresConfig.QueryTimeout))
	}
	if postgresConfig.TxTimeout > 0 {
		opts = append(opts, postgresx.WithTxTimeout(postgresConfig.TxTimeout))
	}

	var poolOpts []postgresx.PoolOption
	if postgresConfig.MaxConns > 0 {
		poolOpts = append(poolOpts, postgresx.WithMaxConns(postgresConfig.MaxConns))
	}
	if postgresConfig.MinConns > 0 {
		poolOpts = append(poolOpts, postgresx.WithMinConns(postgresConfig.MinConns))
	}
	if postgresConfig.MaxConnLifetime > 0 {
		poolOpts = append(poolOpts, postgresx.WithMaxConnLifetime(postgresConfig.MaxConnLifetime))
	}
	if postgresConfig.MaxConnIdleTime > 0 {
		poolOpts = append(poolOpts, postgresx.WithMaxConnIdleTime(postgresConfig.MaxConnIdleTime))
	}
	if postgresConfig.ConnectTimeout > 0 {
		poolOpts = append(poolOpts, postgresx.WithConnectTimeout(postgresConfig.ConnectTimeout))
	}
	if postgresConfig.ApplicationName != "" {
		poolOpts = append(poolOpts, postgresx.WithApplicationName(postgresConfig.ApplicationName))
	}

	if len(poolOpts) > 0 {
		opts = append(opts, postgresx.WithPoolOptions(poolOpts...))
	}

	return opts
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ConsulConfig struct {
	Address string `mapstructure:"addr"`
}

// LoadConfig initializes configuration from flags, environment variables, and files.
func LoadConfig() (*Config, error) {
	defineFlags()
	pflag.Parse()

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return nil, err
	}

	cfg := &Config{}

	configFile := viper.GetString("config_file")
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		viper.OnConfigChange(func(e fsnotify.Event) {
			log.Printf("Config file changed: %s", e.Name)

			newCfg := &Config{}
			if err := viper.Unmarshal(newCfg); err != nil {
				log.Printf("Reload error: unable to decode: %v", err)
				return
			}

			if err := newCfg.validate(); err != nil {
				log.Printf("Reload error: invalid config: %v", err)
				return
			}

			*cfg = *newCfg
			log.Println("Config reloaded successfully")
		})

		viper.WatchConfig()
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %v", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func defineFlags() {
	pflag.String("config_file", "", "Configuration file (YAML, JSON, etc.)")

	pflag.String("service.id", "", "Service ID")
	pflag.String("service.addr", "localhost:8080", "gRPC service address")
	pflag.String("service.http_addr", ":8081", "HTTP service address")
	pflag.String("service.webhook_path", "/wh", "Base path for incoming webhooks")

	pflag.Bool("service.conn.verify_certs", false, "Determine whether to verify certificates")
	pflag.String("service.conn.ca", "", "Server CA certificate path")
	pflag.String("service.conn.key", "", "Server certificate key path")
	pflag.String("service.conn.cert", "", "Server certificate path")
	pflag.String("service.conn.client.ca", "", "Client CA certificate path")
	pflag.String("service.conn.client.key", "", "Client certificate key path")
	pflag.String("service.conn.client.cert", "", "Client certificate path")

	pflag.String("log.level", "info", "Log level")
	pflag.Bool("log.json", false, "Log in JSON format")
	pflag.String("log.file", "", "Log file path")
	pflag.Bool("log.console", true, "Enable console logging")
	pflag.Bool("log.otel", false, "Enable OTEL logging")

	//POSTGRES
	pflag.String("postgres.dsn", "", "Postgres DSN")
	pflag.StringSlice("postgres.replicas", []string{}, "Postgres replica DSNs (comma separated)")
	pflag.Duration("postgres.query_timeout", 0, "Default timeout for queries")
	pflag.Duration("postgres.tx_timeout", 0, "Default timeout for transactions")
	pflag.Int32("postgres.max_conns", 0, "Maximum number of connections in the pool")
	pflag.Int32("postgres.min_conns", 0, "Minimum number of connections in the pool")
	pflag.Duration("postgres.max_conn_lifetime", 0, "Maximum amount of time a connection may exist")
	pflag.Duration("postgres.max_conn_idle_time", 0, "Maximum amount of time a connection may be idle")
	pflag.Duration("postgres.health_check_period", 0, "Period between health checks")
	pflag.Duration("postgres.connect_timeout", 0, "Timeout for establishing a new connection")
	pflag.String("postgres.application_name", "webitel-im-provider", "Application name sent to postgres")
	//POSTGRES

	pflag.String("redis.addr", "localhost:6379", "Redis address")
	pflag.String("redis.password", "", "Redis password")
	pflag.Int("redis.db", 0, "Redis database number")
	pflag.String("consul.addr", "localhost:8500", "Consul address")
	pflag.String("service.secret_key", "", "32-byte secret key for sensitive data encryption")
}

func (c *Config) validate() error {
	if c.Service.ID == "" {
		return fmt.Errorf("config: service.id is required")
	}

	if c.Service.GRPCAddr == "" {
		return fmt.Errorf("config: service.addr is required")
	}

	// Sanitize WebhookPath
	if c.Service.WebhookPath == "" {
		c.Service.WebhookPath = "/wh"
	}
	if !strings.HasPrefix(c.Service.WebhookPath, "/") {
		c.Service.WebhookPath = "/" + c.Service.WebhookPath
	}

	err := validateConnectionConfig(c.Service.Connection)
	if err != nil {
		return err
	}

	if c.Postgres.DSN == "" {
		return errors.InvalidArgument("postgres.dsn is required", errors.WithID("config.config.validate"))
	}

	return nil
}

func validateConnectionConfig(conn ConnectionConfig) error {
	if conn.VerifyCerts {
		if conn.TLS.CA == "" || conn.TLS.Cert == "" || conn.TLS.Key == "" {
			return fmt.Errorf("config: service.conn TLS certificates are required when verify_certs is true")
		}
	}
	return nil
}
