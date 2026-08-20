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
		"poll_interval":"3s",
		"report_interval":"7s",
		"key":"config-secret",
		"rate_limit":5,
		"crypto_key":"public.pem"
	}`)
	prepareFlags(t, "-config", configPath)

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "http://config:8081", cfg.ExternalAddress)
	require.Equal(t, int64(3), cfg.PollInterval)
	require.Equal(t, int64(7), cfg.ReportInterval)
	require.Equal(t, "config-secret", cfg.Key)
	require.Equal(t, 5, cfg.RateLimit)
	require.Equal(t, "public.pem", cfg.CryptoKey)
}

func TestParseFlagsAndEnvironmentOverrideConfig(t *testing.T) {
	configPath := writeConfig(t, `{
		"address":"config:8081",
		"poll_interval":"3s",
		"report_interval":"7s",
		"crypto_key":"config-public.pem"
	}`)
	prepareFlags(t, "-c", configPath, "-a", "flag:8082", "-r", "11")
	t.Setenv("POLL_INTERVAL", "9")
	t.Setenv("CRYPTO_KEY", "env-public.pem")

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "http://flag:8082", cfg.ExternalAddress)
	require.Equal(t, int64(9), cfg.PollInterval)
	require.Equal(t, int64(11), cfg.ReportInterval)
	require.Equal(t, "env-public.pem", cfg.CryptoKey)
}

func TestParseFlagsGetsConfigPathFromEnvironment(t *testing.T) {
	configPath := writeConfig(t, `{"address":"env-config:8083"}`)
	prepareFlags(t)
	t.Setenv("CONFIG", configPath)

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "http://env-config:8083", cfg.ExternalAddress)
}

func TestParseFlagsFindsConfigAfterOtherFlags(t *testing.T) {
	configPath := writeConfig(t, `{"poll_interval":"3s"}`)
	prepareFlags(t, "-a", "flag:8082", "-c", configPath)

	cfg, err := parseFlags()
	require.NoError(t, err)
	require.Equal(t, "http://flag:8082", cfg.ExternalAddress)
	require.Equal(t, int64(3), cfg.PollInterval)
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
	flag.CommandLine = flag.NewFlagSet("agent-test", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = append([]string{"agent-test"}, args...)

	envNames := []string{
		"ADDRESS", "POLL_INTERVAL", "REPORT_INTERVAL", "KEY",
		"RATE_LIMIT", "CRYPTO_KEY", "CONFIG",
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
