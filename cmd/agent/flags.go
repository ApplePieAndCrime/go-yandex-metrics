package main

import (
	"flag"
	"os"
	"strings"

	configutil "github.com/ApplePieAndCrime/go-yandex-metrics/internal/config"
	"github.com/caarlos0/env/v11"
)

// FlagConfig содержит параметры запуска агента.
type FlagConfig struct {
	ExternalAddress string `env:"ADDRESS"`
	PollInterval    int64  `env:"POLL_INTERVAL"`
	ReportInterval  int64  `env:"REPORT_INTERVAL"`
	Key             string `env:"KEY"`
	RateLimit       int    `env:"RATE_LIMIT"`
	CryptoKey       string `env:"CRYPTO_KEY"`
	Config          string
}

type fileConfig struct {
	Address        *string `json:"address"`
	PollInterval   *string `json:"poll_interval"`
	ReportInterval *string `json:"report_interval"`
	Key            *string `json:"key"`
	RateLimit      *int    `json:"rate_limit"`
	CryptoKey      *string `json:"crypto_key"`
}

func parseFlags() (FlagConfig, error) {
	cfg := FlagConfig{
		ExternalAddress: "localhost:8080",
		PollInterval:    2,
		ReportInterval:  10,
		Key:             "",
		RateLimit:       1,
		CryptoKey:       "",
		Config:          "",
	}
	cfg.Config = configutil.FilePath(os.Args[1:], os.Getenv("CONFIG"))
	if cfg.Config != "" {
		if err := applyFileConfig(&cfg); err != nil {
			return FlagConfig{}, err
		}
	}

	if err := env.Parse(&cfg); err != nil {
		return FlagConfig{}, err
	}

	flag.StringVar(&cfg.ExternalAddress, "a", cfg.ExternalAddress, "адрес и порт для старта сервера")
	flag.Int64Var(&cfg.PollInterval, "p", cfg.PollInterval, "интервал опроса в секундах")
	flag.Int64Var(&cfg.ReportInterval, "r", cfg.ReportInterval, "интервал отчетов в секундах")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "ключ для авторизации")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "предел запросов в секунду")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "путь к файлу публичного ключа")
	flag.StringVar(&cfg.Config, "c", cfg.Config, "путь к JSON-файлу конфигурации")
	flag.StringVar(&cfg.Config, "config", cfg.Config, "путь к JSON-файлу конфигурации")

	flag.Parse()

	if !strings.HasPrefix(cfg.ExternalAddress, "http://") &&
		!strings.HasPrefix(cfg.ExternalAddress, "https://") {
		cfg.ExternalAddress = "http://" + cfg.ExternalAddress
	}

	return cfg, nil
}

func applyFileConfig(cfg *FlagConfig) error {
	var fileCfg fileConfig
	if err := configutil.ReadJSON(cfg.Config, &fileCfg); err != nil {
		return err
	}

	if fileCfg.Address != nil {
		cfg.ExternalAddress = *fileCfg.Address
	}
	if fileCfg.PollInterval != nil {
		var err error
		cfg.PollInterval, err = configutil.DurationSeconds("poll_interval", *fileCfg.PollInterval)
		if err != nil {
			return err
		}
	}
	if fileCfg.ReportInterval != nil {
		var err error
		cfg.ReportInterval, err = configutil.DurationSeconds("report_interval", *fileCfg.ReportInterval)
		if err != nil {
			return err
		}
	}
	if fileCfg.Key != nil {
		cfg.Key = *fileCfg.Key
	}
	if fileCfg.RateLimit != nil {
		cfg.RateLimit = *fileCfg.RateLimit
	}
	if fileCfg.CryptoKey != nil {
		cfg.CryptoKey = *fileCfg.CryptoKey
	}

	return nil
}
