package postgresx

import (
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolOption func(*pgxpool.Config)

type OpenOption func(*openConfig)

type openConfig struct {
	poolOpts     []PoolOption
	replicaDSNs  []string
	lbPolicy     LoadBalancingPolicy
	queryTimeout time.Duration
	txTimeout    time.Duration
}

func defaultOpenConfig() *openConfig {
	return &openConfig{
		lbPolicy:     RoundRobin,
		poolOpts:     DefaultPoolOptions(),
		queryTimeout: 0,
		txTimeout:    0,
	}
}

func WithPoolOptions(opts ...PoolOption) OpenOption {
	return func(c *openConfig) {
		c.poolOpts = append(c.poolOpts, opts...)
	}
}

func WithReplicas(dsns ...string) OpenOption {
	return func(c *openConfig) {
		c.replicaDSNs = append(c.replicaDSNs, dsns...)
	}
}

func WithLoadBalancer(policy LoadBalancingPolicy) OpenOption {
	return func(c *openConfig) {
		c.lbPolicy = policy
	}
}

func WithQueryTimeout(d time.Duration) OpenOption {
	return func(c *openConfig) {
		c.queryTimeout = d
	}
}

func WithTxTimeout(d time.Duration) OpenOption {
	return func(c *openConfig) {
		c.txTimeout = d
	}
}

func WithMaxConns(n int32) PoolOption {
	return func(c *pgxpool.Config) { c.MaxConns = n }
}

func WithMinConns(n int32) PoolOption {
	return func(c *pgxpool.Config) { c.MinConns = n }
}

func WithAutoMaxConns(multiplier int32) PoolOption {
	return func(c *pgxpool.Config) {
		c.MaxConns = int32(runtime.GOMAXPROCS(0)) * multiplier
	}
}

func WithMaxConnLifetime(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) { c.MaxConnLifetime = d }
}

func WithMaxConnLifetimeJitter(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) { c.MaxConnLifetimeJitter = d }
}

func WithMaxConnIdleTime(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) { c.MaxConnIdleTime = d }
}

func WithHealthCheckPeriod(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) { c.HealthCheckPeriod = d }
}

func WithConnectTimeout(d time.Duration) PoolOption {
	return func(c *pgxpool.Config) { c.ConnConfig.ConnectTimeout = d }
}

func WithApplicationName(name string) PoolOption {
	return func(c *pgxpool.Config) {
		c.ConnConfig.RuntimeParams["application_name"] = name
	}
}

func DefaultPoolOptions() []PoolOption {
	return []PoolOption{
		WithAutoMaxConns(4),
		WithMinConns(2),
		WithMaxConnLifetime(30 * time.Minute),
		WithMaxConnLifetimeJitter(5 * time.Minute),
		WithMaxConnIdleTime(10 * time.Minute),
		WithHealthCheckPeriod(30 * time.Second),
		WithConnectTimeout(5 * time.Second),
	}
}
