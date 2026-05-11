package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
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

func migrateDb(flagConfig FlagConfig, loggerSugar zap.SugaredLogger) {
	if flagConfig.DatabaseDsn == "" {
		return
	}
	db, err := sql.Open("pgx", flagConfig.DatabaseDsn)
	if err != nil {
		loggerSugar.Fatalf("database connection error: %v", err)
		return
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		loggerSugar.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		loggerSugar.Fatal(err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		loggerSugar.Fatalf("Migration up failed: %v", err)
	}
	loggerSugar.Infoln("Migration up completed successfully")
}

func RunServer(flagConfig FlagConfig, loggerSugar zap.SugaredLogger) error {
	migrateDb(flagConfig, loggerSugar)

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
