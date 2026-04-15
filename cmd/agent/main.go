package main

import (
	internal_agent "github.com/ApplePieAndCrime/go-yandex-metrics/internal/agent"
)

func main() {
	parseFlags()
	internal_agent.RunAgent(&flagExternalAddress, &flagPollInterval, &flagReportInterval)
}
