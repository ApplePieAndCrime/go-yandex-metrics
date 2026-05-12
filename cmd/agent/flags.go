package main

import (
	"flag"
	"strings"

	"github.com/caarlos0/env/v11"
)

type FlagConfig struct {
	ExternalAddress string `env:"ADDRESS"`
	PollInterval    int64  `env:"POLL_INTERVAL"`
	ReportInterval  int64  `env:"REPORT_INTERVAL"`
}

func parseFlags() FlagConfig {
	cfg := FlagConfig{
		ExternalAddress: "localhost:8080",
		PollInterval:    2,
		ReportInterval:  10,
	}
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}

	flag.StringVar(&cfg.ExternalAddress, "a", cfg.ExternalAddress, "address and port to send HTTP requests")
	flag.Int64Var(&cfg.PollInterval, "p", cfg.PollInterval, "poll interval in seconds")
	flag.Int64Var(&cfg.ReportInterval, "r", cfg.ReportInterval, "report interval in seconds")

	flag.Parse()

	if !strings.HasPrefix(cfg.ExternalAddress, "http://") &&
		!strings.HasPrefix(cfg.ExternalAddress, "https://") {
		cfg.ExternalAddress = "http://" + cfg.ExternalAddress
	}

	return cfg
}
