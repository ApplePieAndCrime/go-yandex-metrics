package main

import (
	"flag"
	"strings"

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
}

func parseFlags() (FlagConfig, error) {
	cfg := FlagConfig{
		ExternalAddress: "localhost:8080",
		PollInterval:    2,
		ReportInterval:  10,
		Key:             "",
		RateLimit:       1,
		CryptoKey:       "",
	}
	err := env.Parse(&cfg)
	if err != nil {
		return FlagConfig{}, err
	}

	flag.StringVar(&cfg.ExternalAddress, "a", cfg.ExternalAddress, "адрес и порт для старта сервера")
	flag.Int64Var(&cfg.PollInterval, "p", cfg.PollInterval, "интервал опроса в секундах")
	flag.Int64Var(&cfg.ReportInterval, "r", cfg.ReportInterval, "интервал отчетов в секундах")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "ключ для авторизации")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "предел запросов в секунду")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "путь к файлу публичного ключа")

	flag.Parse()

	if !strings.HasPrefix(cfg.ExternalAddress, "http://") &&
		!strings.HasPrefix(cfg.ExternalAddress, "https://") {
		cfg.ExternalAddress = "http://" + cfg.ExternalAddress
	}

	return cfg, nil
}
