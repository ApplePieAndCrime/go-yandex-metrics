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
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

func migrateDb(flagConfig FlagConfig, loggerSugar zap.SugaredLogger) error {
	if flagConfig.DatabaseDsn == "" {
		return nil
	}
	db, err := sql.Open("pgx", flagConfig.DatabaseDsn)
	if err != nil {
		loggerSugar.Fatalf("database connection error: %v", err)
		return err
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

	return nil
}

func RunServer(flagConfig FlagConfig, loggerSugar zap.SugaredLogger) error {

	var storage repository.Storage
	var db *sql.DB
	var err error

	if flagConfig.DatabaseDsn != "" {
		db, err = sql.Open("pgx", flagConfig.DatabaseDsn)
		if err != nil {
			return err
		}
		if err := migrateDb(flagConfig, loggerSugar); err != nil {
			return err
		}

		storage = repository.NewPostgresStorage(db)
	} else {
		storage = repository.NewMemoryStorage()
	}

	if flagConfig.Key != "" {
		loggerSugar.Infoln("Using key for authentication")
	}

	services := service.NewService(storage)
	handlers := handler.NewHandler(services, loggerSugar, db, flagConfig.Key)

	routes := handlers.InitRoutes()

	if flagConfig.DatabaseDsn == "" {
		errCh := server.SaveMetricsToFile(*services, flagConfig.Interval, flagConfig.StoragePath, flagConfig.IsRestore)
		go func() {
			if err := <-errCh; err != nil {
				log.Println("save metrics error:", err)
			}
		}()
	}

	log.Println("Server is running on address:", flagConfig.RunAddress)
	return http.ListenAndServe(flagConfig.RunAddress, routes)
}
