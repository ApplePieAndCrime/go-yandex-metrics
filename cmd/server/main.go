package main

import (
	"fmt"
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
)

func main() {
	logger.LoggerInitialize()
	parseFlags()

	err := RunServer()
	if err != nil {
		panic(err)
	}
}

func RunServer() error {
	repositoryResponse := repository.Init()
	routes := handler.Init(repositoryResponse)

	fmt.Println("Server is running on address: " + flagRunAddress)

	err := http.ListenAndServe(flagRunAddress, routes)
	return err
}
