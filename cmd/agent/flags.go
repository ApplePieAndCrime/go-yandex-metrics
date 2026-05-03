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
	var cfg FlagConfig
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}

	if cfg.ExternalAddress != "" {
		cfg.ExternalAddress = cfg.ExternalAddress
	} else {
		flag.StringVar(&cfg.ExternalAddress, "a", "localhost:8080", "address and port to send HTTP requests")
	}

	if cfg.PollInterval != 0 {
		cfg.PollInterval = cfg.PollInterval
	} else {
		flag.Int64Var(&cfg.PollInterval, "p", 2, "poll interval in seconds")
	}

	if cfg.ReportInterval != 0 {
		cfg.ReportInterval = cfg.ReportInterval
	} else {
		flag.Int64Var(&cfg.ReportInterval, "r", 10, "report interval in seconds")
	}

	flag.Parse()

	if !strings.HasPrefix(cfg.ExternalAddress, "http://") &&
		!strings.HasPrefix(cfg.ExternalAddress, "https://") {
		cfg.ExternalAddress = "http://" + cfg.ExternalAddress
	}

	return cfg
}
