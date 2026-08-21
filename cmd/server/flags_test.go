package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFlagsFromConfig(t *testing.T) {
	configPath := writeConfig(t, `{
		"address":"config:8081",
		"restore":false,
		"store_interval":"4s",
		"store_file":"config.db",
		"database_dsn":"config-dsn",
		"key":"config-secret",
		"audit_file":"audit.log",
		"audit_url":"http://audit.local",
		"crypto_key":"private.pem",
		"trusted_subnet":"192.168.1.0/24",
		"grpc_address":"config-grpc:3200",
		"grpc_cert_file":"config-server.crt",
		"grpc_key_file":"config-server.key"
	}`)
	prepareFlags(t, "-config", configPath)

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "config:8081", cfg.RunAddress)
	require.False(t, cfg.IsRestore)
	require.Equal(t, int64(4), cfg.Interval)
	require.Equal(t, "config.db", cfg.StoragePath)
	require.Equal(t, "config-dsn", cfg.DatabaseDsn)
	require.Equal(t, "config-secret", cfg.Key)
	require.Equal(t, "audit.log", cfg.AuditFile)
	require.Equal(t, "http://audit.local", cfg.AuditUrl)
	require.Equal(t, "private.pem", cfg.CryptoKey)
	require.Equal(t, "192.168.1.0/24", cfg.TrustedSubnet)
	require.Equal(t, "config-grpc:3200", cfg.GRPCAddress)
	require.Equal(t, "config-server.crt", cfg.GRPCCertFile)
	require.Equal(t, "config-server.key", cfg.GRPCKeyFile)
}

func TestParseFlagsAndEnvironmentOverrideConfig(t *testing.T) {
	configPath := writeConfig(t, `{
		"address":"config:8081",
		"restore":false,
		"store_interval":"4s",
		"store_file":"config.db",
		"crypto_key":"config-private.pem",
		"trusted_subnet":"10.0.0.0/8"
	}`)
	prepareFlags(t, "-c", configPath, "-a", "flag:8082", "-i", "11", "-t", "172.16.0.0/12", "-g", "flag-grpc:3201", "-grpc-cert", "flag-server.crt", "-grpc-key", "flag-server.key")
	t.Setenv("RESTORE", "true")
	t.Setenv("STORE_FILE", "env.db")
	t.Setenv("CRYPTO_KEY", "env-private.pem")
	t.Setenv("TRUSTED_SUBNET", "192.168.0.0/16")
	t.Setenv("GRPC_ADDRESS", "env-grpc:3202")
	t.Setenv("GRPC_CERT_FILE", "env-server.crt")
	t.Setenv("GRPC_KEY_FILE", "env-server.key")

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "flag:8082", cfg.RunAddress)
	require.True(t, cfg.IsRestore)
	require.Equal(t, int64(11), cfg.Interval)
	require.Equal(t, "env.db", cfg.StoragePath)
	require.Equal(t, "env-private.pem", cfg.CryptoKey)
	require.Equal(t, "172.16.0.0/12", cfg.TrustedSubnet)
	require.Equal(t, "flag-grpc:3201", cfg.GRPCAddress)
	require.Equal(t, "flag-server.crt", cfg.GRPCCertFile)
	require.Equal(t, "flag-server.key", cfg.GRPCKeyFile)
}

func TestParseFlagsGetsConfigPathFromEnvironment(t *testing.T) {
	configPath := writeConfig(t, `{"address":"env-config:8083"}`)
	prepareFlags(t)
	t.Setenv("CONFIG", configPath)

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "env-config:8083", cfg.RunAddress)
}

func TestParseFlagsFindsConfigAfterOtherFlags(t *testing.T) {
	configPath := writeConfig(t, `{"store_interval":"4s"}`)
	prepareFlags(t, "-a", "flag:8082", "-config", configPath)

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "flag:8082", cfg.RunAddress)
	require.Equal(t, int64(4), cfg.Interval)
}

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

func prepareFlags(t *testing.T, args ...string) {
	t.Helper()

	oldFlags := flag.CommandLine
	oldArgs := os.Args
	flag.CommandLine = flag.NewFlagSet("server-test", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = append([]string{"server-test"}, args...)

	envNames := []string{
		"ADDRESS", "STORE_INTERVAL", "FILE_STORAGE_PATH", "STORE_FILE", "RESTORE",
		"DATABASE_DSN", "KEY", "AUDIT_FILE", "AUDIT_URL", "CRYPTO_KEY", "TRUSTED_SUBNET", "GRPC_ADDRESS", "GRPC_CERT_FILE", "GRPC_KEY_FILE", "CONFIG",
	}
	type envValue struct {
		value  string
		exists bool
	}
	oldEnv := make(map[string]envValue, len(envNames))
	for _, name := range envNames {
		value, exists := os.LookupEnv(name)
		oldEnv[name] = envValue{value: value, exists: exists}
		require.NoError(t, os.Unsetenv(name))
	}

	t.Cleanup(func() {
		flag.CommandLine = oldFlags
		os.Args = oldArgs
		for name, old := range oldEnv {
			if old.exists {
				_ = os.Setenv(name, old.value)
			} else {
				_ = os.Unsetenv(name)
			}
		}
	})
}
