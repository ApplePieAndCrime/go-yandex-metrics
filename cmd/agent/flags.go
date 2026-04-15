package main

import (
	"flag"
	"strings"
)

var flagExternalAddress string
var flagPollInterval int64
var flagReportInterval int64

func parseFlags() {
	flag.StringVar(&flagExternalAddress, "a", "localhost:8080", "address and port to send HTTP requests")
	flag.Int64Var(&flagPollInterval, "p", 2, "poll interval in seconds")
	flag.Int64Var(&flagReportInterval, "r", 10, "report interval in seconds")

	flag.Parse()

	if !strings.HasPrefix(flagExternalAddress, "http://") &&
		!strings.HasPrefix(flagExternalAddress, "https://") {
		flagExternalAddress = "http://" + flagExternalAddress
	}
}
