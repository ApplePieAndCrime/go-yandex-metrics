package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	internal_agent "github.com/ApplePieAndCrime/go-yandex-metrics/internal/agent"
)

func main() {
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
