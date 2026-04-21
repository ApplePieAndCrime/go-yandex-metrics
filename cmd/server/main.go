package main

import (
	"fmt"
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
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
	repos := repository.NewRepository()
	services := service.NewService(repos)
	handlers := handler.NewHandler(services)

	routes := handlers.InitRoutes()

	fmt.Println("Server is running on address: " + flagRunAddress)

	err := http.ListenAndServe(flagRunAddress, routes)
	return err
}
