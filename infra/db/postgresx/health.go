package postgresx

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthStatus struct {
	DSN     string        `json:"dsn"`
	Role    string        `json:"role"`
	Healthy bool          `json:"healthy"`
	Latency time.Duration `json:"latency_ms"`
	Error   string        `json:"error,omitempty"`
	Stats   PoolStats     `json:"stats"`
}

type PoolStats struct {
	TotalConns        int32 `json:"total_conns"`
	IdleConns         int32 `json:"idle_conns"`
	AcquiredConns     int32 `json:"acquired_conns"`
	MaxConns          int32 `json:"max_conns"`
	NewConnsCount     int64 `json:"new_conns_count"`
	EmptyAcquireCount int64 `json:"empty_acquire_count"`
}

type Report struct {
	Healthy   bool           `json:"healthy"`
	Pools     []HealthStatus `json:"pools"`
	CheckedAt time.Time      `json:"checked_at"`
}

type Checker struct {
	primary  *pgxpool.Pool
	replicas []*pgxpool.Pool
	timeout  time.Duration
}

func NewChecker(d DB, timeout time.Duration) *Checker {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &Checker{
		primary:  d.PrimaryPool(),
		replicas: d.ReplicaPools(),
		timeout:  timeout,
	}
}

func (hc *Checker) Check(ctx context.Context) Report {
	ctx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	total := 1 + len(hc.replicas)
	statuses := make([]HealthStatus, total)
	var wg sync.WaitGroup
	wg.Add(total)

	check := func(i int, pool *pgxpool.Pool, role string) {
		defer wg.Done()
		statuses[i] = pingPool(ctx, pool, role)
	}

	go check(0, hc.primary, "primary")
	for i, r := range hc.replicas {
		go check(i+1, r, "replica")
	}
	wg.Wait()

	healthy := statuses[0].Healthy
	return Report{
		Healthy:   healthy,
		Pools:     statuses,
		CheckedAt: time.Now().UTC(),
	}
}

func pingPool(ctx context.Context, pool *pgxpool.Pool, role string) HealthStatus {
	stat := pool.Stat()
	s := HealthStatus{
		DSN:  maskDSN(pool.Config().ConnConfig.ConnString()),
		Role: role,
		Stats: PoolStats{
			TotalConns:        stat.TotalConns(),
			IdleConns:         stat.IdleConns(),
			AcquiredConns:     stat.AcquiredConns(),
			MaxConns:          stat.MaxConns(),
			NewConnsCount:     stat.NewConnsCount(),
			EmptyAcquireCount: stat.EmptyAcquireCount(),
		},
	}

	start := time.Now()
	err := pool.Ping(ctx)
	s.Latency = time.Since(start)
	s.Healthy = err == nil
	if err != nil {
		s.Error = err.Error()
	}
	return s
}

func maskDSN(dsn string) string {
	const mask = "password=***"
	for _, field := range []string{"password="} {
		if idx := strings.Index(dsn, field); idx != -1 {
			end := strings.IndexByte(dsn[idx+len(field):], ' ')
			if end == -1 {
				dsn = dsn[:idx] + mask
			} else {
				dsn = dsn[:idx] + mask + dsn[idx+len(field)+end:]
			}
		}
	}
	return dsn
}
