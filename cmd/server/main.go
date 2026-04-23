package main

import (
	"fmt"
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
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

	// Запускаем фоновое сохранение метрик ДО HTTP-сервера
	go server.SaveMetricsToFile(*services, flagInterval, flagStoragePath, flagRestore)

	err := http.ListenAndServe(flagRunAddress, routes)
	return err
}
