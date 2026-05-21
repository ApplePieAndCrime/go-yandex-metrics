package main

import (
	"log"

	internal_agent "github.com/ApplePieAndCrime/go-yandex-metrics/internal/agent"
)

func main() {
	flagConfig := parseFlags()
	log.Println("AGENT CONFIG ", flagConfig)

	err := internal_agent.RunAgent(&flagConfig.ExternalAddress, &flagConfig.PollInterval, &flagConfig.ReportInterval, &flagConfig.Key)
	if err != nil {
		panic(err)
	}
}
