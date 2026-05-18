//go:build integration

package postgres

// Integration tests for the postgres store layer.
// Run with: go test ./internal/store/postgres/... -tags integration -v
//
// The tests spin up a real PostgreSQL container via testcontainers-go,
// run all migrations, and execute every store operation against a live DB.
// Each test uses t.Cleanup to delete its own data so tests are order-independent.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tc "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/webitel/im-providers-service/config"
	"github.com/webitel/im-providers-service/internal/store"
	"github.com/webitel/im-providers-service/internal/store/lru"
	"github.com/webitel/im-providers-service/pkg/crypto"
)

// shared test-wide fixtures, populated by TestMain.
var (
	testPool   *pgxpool.Pool
	testCrypto crypto.Encryptor
	testCache  store.GateCache
	testCfg    *config.Config
)

// aes-256 key (32 bytes) used only in tests.
const testAESKey = "test-secret-key-32-bytes-long!!!"

func TestMain(m *testing.M) {
	ctx := context.Background()

	pgCtr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = pgCtr.Terminate(ctx) }()

	dsn, err := pgCtr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get connection string: %v\n", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "run migrations: %v\n", err)
		os.Exit(1)
	}

	cache, err := lru.NewLRUCache(128)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create lru cache: %v\n", err)
		os.Exit(1)
	}

	testPool = pool
	testCrypto = crypto.NewAESGCM(testAESKey)
	testCache = cache
	testCfg = &config.Config{}

	os.Exit(m.Run())
}

// runMigrations creates the uuidv7() shim and runs every migration file in order.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// uuidv7() is not built-in to PostgreSQL — provide a v4 shim for tests.
	const uuidShim = `
		CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid AS $$
			SELECT gen_random_uuid();
		$$ LANGUAGE sql;`

	if _, err := pool.Exec(ctx, uuidShim); err != nil {
		return fmt.Errorf("uuidv7 shim: %w", err)
	}

	migrationsDir := migrationDir()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	for _, f := range files {
		sql, err := extractUpSQL(f)
		if err != nil {
			return fmt.Errorf("parse %s: %w", f, err)
		}
		if sql == "" {
			continue
		}
		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
	}
	return nil
}

// migrationDir returns the absolute path to migrations/ relative to this file.
func migrationDir() string {
	_, file, _, _ := runtime.Caller(0)
	// file = .../internal/store/postgres/testmain_test.go
	// migrations/ is at the repo root
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
	return root
}

// extractUpSQL reads a goose-formatted migration file and returns only the Up SQL,
// stripping all goose annotation lines.
func extractUpSQL(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(raw)

	upIdx := strings.Index(content, "-- +goose Up")
	if upIdx < 0 {
		return "", nil
	}
	content = content[upIdx+len("-- +goose Up"):]

	if downIdx := strings.Index(content, "-- +goose Down"); downIdx >= 0 {
		content = content[:downIdx]
	}

	// Strip goose annotation lines; keep everything else.
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- +goose") {
			continue
		}
		lines = append(lines, line)
	}

	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}
