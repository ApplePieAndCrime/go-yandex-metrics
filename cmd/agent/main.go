package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	internal_agent "github.com/ApplePieAndCrime/go-yandex-metrics/internal/agent"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	printBuildInfo()

	flagConfig, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err = internal_agent.RunAgent(
		ctx,
		flagConfig.ExternalAddress,
		flagConfig.PollInterval,
		flagConfig.ReportInterval,
		flagConfig.Key,
		flagConfig.RateLimit,
	)

	if err != nil {
		log.Fatal(err)
	}
}

func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}
	date := buildDate
	if date == "" {
		date = "N/A"
	}
	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", version, date, commit)
}
