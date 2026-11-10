package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"

	"github.com/webitel/webitel-go-kit/appconfig/secret"

	"github.com/webitel/im-providers-service/config"
)

// startAsSystemdUnit makes Load* see what a systemd unit with
// LoadCredential*= sees: no secret env vars, only $CREDENTIALS_DIRECTORY
// holding the given credentials. It also resets the global flag set, since
// the loaders register into pflag.CommandLine and parse os.Args.
func startAsSystemdUnit(t *testing.T, creds map[string]string) string {
	t.Helper()

	for _, env := range []string{
		"CONFIG_FILE",
		"POSTGRES_DSN", "POSTGRES_DSN_FILE",
		"REDIS_PASSWORD", "REDIS_PASSWORD_FILE",
		"SERVICE_SECRET_KEY", "SERVICE_SECRET_KEY_FILE",
		"SERVICE_CONN_KEY", "SERVICE_CONN_CLIENT_KEY",
	} {
		t.Setenv(env, "")
		os.Unsetenv(env)
	}

	dir := t.TempDir()
	for name, value := range creds {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o400); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("CREDENTIALS_DIRECTORY", dir)

	args, flags := os.Args, pflag.CommandLine
	os.Args = []string{"server"}
	pflag.CommandLine = pflag.NewFlagSet("server", pflag.ContinueOnError)

	t.Cleanup(func() { os.Args, pflag.CommandLine = args, flags })

	return dir
}

// LoadConfig uses a package-level loader, so it is exercised once per test
// binary; the conflict case goes through LoadMigrateConfig instead.
func TestLoadConfigSecretsFromCredentials(t *testing.T) {
	dir := startAsSystemdUnit(t, map[string]string{
		"postgres.dsn":       "postgres://db.example.com/im_test",
		"redis.password":     "test-redis-password",
		"service.secret_key": "changeme",
		"service.conn.key":   "test-only-key",
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	want := map[string]string{
		"postgres.dsn":       "postgres://db.example.com/im_test",
		"redis.password":     "test-redis-password",
		"service.secret_key": "changeme",
		"service.conn.key":   filepath.Join(dir, "service.conn.key"),
	}

	for name, got := range map[string]string{
		"postgres.dsn":       cfg.Postgres.DSN,
		"redis.password":     cfg.Redis.Password,
		"service.secret_key": cfg.Service.SecretKey,
		"service.conn.key":   cfg.Service.Connection.Key,
	} {
		if got != want[name] {
			t.Errorf("%s: got %q, want %q", name, got, want[name])
		}
	}
}

func TestLoadMigrateConfigDSNFromCredential(t *testing.T) {
	startAsSystemdUnit(t, map[string]string{"postgres.dsn": "postgres://db.example.com/im_test"})

	cfg, err := config.LoadMigrateConfig()
	if err != nil {
		t.Fatalf("LoadMigrateConfig: %v", err)
	}

	if cfg.Postgres.DSN != "postgres://db.example.com/im_test" {
		t.Errorf("postgres.dsn: got %q", cfg.Postgres.DSN)
	}
}

func TestLoadMigrateConfigPlainDSNShadowingCredentialFails(t *testing.T) {
	startAsSystemdUnit(t, map[string]string{"postgres.dsn": "postgres://db.example.com/new"})
	t.Setenv("POSTGRES_DSN", "postgres://db.example.com/old")

	_, err := config.LoadMigrateConfig()
	if !errors.Is(err, secret.ErrConflict) {
		t.Fatalf("want secret.ErrConflict, got %v", err)
	}
}
