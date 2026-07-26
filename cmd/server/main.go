package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/audit"
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

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	printBuildInfo()

	loggerSugar := logger.LoggerInitialize()
	flagConfig, err := parseFlags()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err = RunServer(ctx, *flagConfig, loggerSugar)
	if err != nil {
		log.Fatal(err)
	}
}

func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}
	date := buildDate
	if date == "" {
		date = "N/A"
	}
	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", version, date, commit)
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

// RunServer настраивает зависимости и запускает HTTP-сервер метрик.
func RunServer(ctx context.Context, flagConfig FlagConfig, loggerSugar zap.SugaredLogger) error {

	var storage repository.Storage
	var db *sql.DB
	var err error

	if flagConfig.DatabaseDsn != "" {
		db, err = sql.Open("pgx", flagConfig.DatabaseDsn)
		if err != nil {
			return err
		}
		defer db.Close()
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

	auditPublisher := audit.NewManager(
		audit.NewFileObserver(flagConfig.AuditFile),
		audit.NewHTTPObserver(flagConfig.AuditUrl, http.DefaultClient),
	)
	defer func() {
		if err := auditPublisher.Close(); err != nil {
			loggerSugar.Errorw("failed to close audit publisher", "error", err)
		}
	}()

	services := service.NewService(storage)
	handlers := handler.NewHandler(services, loggerSugar, db, flagConfig.Key, auditPublisher)

	routes := handlers.InitRoutes()

	if flagConfig.DatabaseDsn == "" {
		errCh := server.SaveMetricsToFile(*services, flagConfig.Interval, flagConfig.StoragePath, flagConfig.IsRestore)
		go func() {
			if err := <-errCh; err != nil {
				log.Println("save metrics error:", err)
			}
		}()
	}

	httpServer := &http.Server{
		Addr:    flagConfig.RunAddress,
		Handler: routes,
	}
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()

	log.Println("Server is running on address:", flagConfig.RunAddress)
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}
