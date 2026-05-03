package main

import (
	"log"
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"go.uber.org/zap"
)

func main() {
	loggerSugar := logger.LoggerInitialize()
	flagConfig, err := parseFlags()

	log.Println("SERVER CONFIG ", flagConfig)

	err = RunServer(*flagConfig, loggerSugar)
	if err != nil {
		log.Fatal(err)
	}
}

func RunServer(flagConfig FlagConfig, loggerSugar zap.SugaredLogger) error {
	repos := repository.NewRepository()
	services := service.NewService(repos)
	handlers := handler.NewHandler(services, loggerSugar, flagConfig.DatabaseDsn)

	routes := handlers.InitRoutes()

	log.Println("Server is running on address: " + flagConfig.RunAddress)

	// Запускаем фоновое сохранение метрик ДО HTTP-сервера
	errCh := server.SaveMetricsToFile(*services, flagConfig.Interval, flagConfig.StoragePath, flagConfig.IsRestore)
	go func() {
		if err := <-errCh; err != nil {
			log.Println("save metrics error:", err)
		}
	}()

	err := http.ListenAndServe(flagConfig.RunAddress, routes)
	return err
}
