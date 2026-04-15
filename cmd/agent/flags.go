package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env/v6"
)

var flagExternalAddress string
var flagPollInterval int64
var flagReportInterval int64

type Config struct {
	ExternalAddress string `env:"ADDRESS"`
	PollInterval    int64  `env:"POLL_INTERVAL"`
	ReportInterval  int64  `env:"REPORT_INTERVAL"`
}

func parseFlags() {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("CONFIG ", cfg)

	if cfg.ExternalAddress != "" {
		flagExternalAddress = cfg.ExternalAddress
	} else {
		flag.StringVar(&flagExternalAddress, "a", "localhost:8080", "address and port to send HTTP requests")
	}

	if cfg.PollInterval != 0 {
		flagPollInterval = cfg.PollInterval
	} else {
		flag.Int64Var(&flagPollInterval, "p", 2, "poll interval in seconds")
	}

	if cfg.ReportInterval != 0 {
		flagReportInterval = cfg.ReportInterval
	} else {
		flag.Int64Var(&flagReportInterval, "r", 10, "report interval in seconds")
	}

	flag.Parse()

	if !strings.HasPrefix(flagExternalAddress, "http://") &&
		!strings.HasPrefix(flagExternalAddress, "https://") {
		flagExternalAddress = "http://" + flagExternalAddress
	}

	fmt.Println("ExternalAddress: ", flagExternalAddress)
	fmt.Println("PollInterval: ", flagPollInterval)
	fmt.Println("ReportInterval: ", flagReportInterval)
}
