package main

import (
	"context"
	"os/signal"
	"syscall"

	internal_agent "github.com/ApplePieAndCrime/go-yandex-metrics/internal/agent"
)

func main() {
	flagConfig := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err := internal_agent.RunAgent(
		ctx,
		flagConfig.ExternalAddress,
		flagConfig.PollInterval,
		flagConfig.ReportInterval,
		flagConfig.Key,
		flagConfig.RateLimit,
	)

	if err != nil {
		panic(err)
	}
}
