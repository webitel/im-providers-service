package postgresx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/webitel/webitel-go-kit/pkg/errors"
)

func New(ctx context.Context, primaryDSN string, opts ...OpenOption) (DB, error) {
	cfg := defaultOpenConfig()
	for _, o := range opts {
		o(cfg)
	}

	primary, err := buildPool(ctx, primaryDSN, cfg.poolOpts)
	if err != nil {
		return nil, errors.Internal("open primary database", errors.WithCause(err), errors.WithID("postgresx.open.new"))
	}

	replicas := make([]*pgxpool.Pool, 0, len(cfg.replicaDSNs))
	for i, dsn := range cfg.replicaDSNs {
		r, err := buildPool(ctx, dsn, cfg.poolOpts)
		if err != nil {
			primary.Close()
			for _, rr := range replicas {
				rr.Close()
			}
			return nil, errors.Internal("open replica connection", errors.WithCause(err), errors.WithID("postgresx.open.new"), errors.WithValue("replica idx", i))
		}
		replicas = append(replicas, r)
	}

	return &database{
		primary:      primary,
		replicas:     replicas,
		lb:           newLoadBalancer(cfg.lbPolicy),
		queryTimeout: cfg.queryTimeout,
		txTimeout:    cfg.txTimeout,
	}, nil
}

func MustNew(ctx context.Context, primaryDSN string, opts ...OpenOption) DB {
	d, err := New(ctx, primaryDSN, opts...)
	if err != nil {
		panic(err)
	}
	return d
}

func buildPool(ctx context.Context, dsn string, opts []PoolOption) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.InvalidArgument("parse dsn", errors.WithCause(err), errors.WithID("postgresx.open.build_pool"))
	}
	for _, o := range opts {
		o(cfg)
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}
