package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	internal_agent "github.com/ApplePieAndCrime/go-yandex-metrics/internal/agent"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	printBuildInfo()

	flagConfig, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	err = internal_agent.RunAgent(
		ctx,
		flagConfig.ExternalAddress,
		flagConfig.PollInterval,
		flagConfig.ReportInterval,
		flagConfig.Key,
		flagConfig.RateLimit,
		flagConfig.CryptoKey,
		internal_agent.GRPCConfig{
			Address:    flagConfig.GRPCAddress,
			CertFile:   flagConfig.GRPCCertFile,
			ServerName: flagConfig.GRPCServerName,
		},
	)

	if err != nil {
		log.Fatal(err)
	}
}

func printBuildInfo() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
}
