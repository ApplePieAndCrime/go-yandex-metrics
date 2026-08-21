package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/audit"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/cryptoutil"
	grpcserver "github.com/ApplePieAndCrime/go-yandex-metrics/internal/grpcserver"
	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
	pb "github.com/ApplePieAndCrime/go-yandex-metrics/internal/proto"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/repository"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/server"
	logger "github.com/ApplePieAndCrime/go-yandex-metrics/internal/server/logger"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/service"
	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/tlsutil"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	printBuildInfo()

	loggerSugar := logger.LoggerInitialize()
	flagConfig, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer stop()

	err = RunServer(ctx, *flagConfig, loggerSugar)
	if err != nil {
		log.Fatal(err)
	}
}

func printBuildInfo() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
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
	var trustedSubnet *net.IPNet
	if flagConfig.TrustedSubnet != "" {
		_, parsedSubnet, err := net.ParseCIDR(flagConfig.TrustedSubnet)
		if err != nil {
			return fmt.Errorf("parse trusted subnet: %w", err)
		}
		trustedSubnet = parsedSubnet
	}

	var privateKey *rsa.PrivateKey
	if flagConfig.CryptoKey != "" {
		var err error
		privateKey, err = cryptoutil.LoadPrivateKey(flagConfig.CryptoKey)
		if err != nil {
			return fmt.Errorf("load server crypto key: %w", err)
		}
		loggerSugar.Infoln("Using private key for request decryption")
	}

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
	handlers := handler.NewHandler(services, loggerSugar, db, flagConfig.Key, auditPublisher, privateKey, trustedSubnet)

	routes := handlers.InitRoutes()

	var grpcServer *grpc.Server
	var grpcListener net.Listener
	if flagConfig.GRPCAddress != "" {
		if flagConfig.GRPCCertFile == "" || flagConfig.GRPCKeyFile == "" {
			return fmt.Errorf("gRPC TLS certificate and private key are required")
		}
		transportCredentials, err := tlsutil.LoadServerCredentials(
			flagConfig.GRPCCertFile,
			flagConfig.GRPCKeyFile,
		)
		if err != nil {
			return fmt.Errorf("load gRPC server credentials: %w", err)
		}
		grpcListener, err = net.Listen("tcp", flagConfig.GRPCAddress)
		if err != nil {
			return fmt.Errorf("listen gRPC address: %w", err)
		}
		grpcServer = grpc.NewServer(
			grpc.Creds(transportCredentials),
			grpc.UnaryInterceptor(grpcserver.TrustedSubnetInterceptor(trustedSubnet)),
		)
		pb.RegisterMetricsServer(grpcServer, grpcserver.NewMetricsServer(services, &loggerSugar))
	}

	var (
		stopFileStorage context.CancelFunc
		fileStorageDone <-chan error
	)
	if flagConfig.DatabaseDsn == "" {
		fileStorageCtx, cancel := context.WithCancel(context.Background())
		stopFileStorage = cancel
		fileStorageDone = server.SaveMetricsToFileWithContext(
			fileStorageCtx,
			*services,
			flagConfig.Interval,
			flagConfig.StoragePath,
			flagConfig.IsRestore,
		)
	}

	httpServer := &http.Server{
		Addr:    flagConfig.RunAddress,
		Handler: routes,
	}
	serverErrors := make(chan error, 2)
	go func() {
		serverErrors <- httpServer.ListenAndServe()
	}()
	if grpcServer != nil {
		go func() {
			serverErrors <- grpcServer.Serve(grpcListener)
		}()
	}

	log.Println("Server is running on address:", flagConfig.RunAddress)
	if grpcServer != nil {
		log.Println("gRPC server is running on address:", flagConfig.GRPCAddress)
	}
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) || errors.Is(err, grpc.ErrServerStopped) {
			err = nil
		}
		shutdownErr := httpServer.Shutdown(context.Background())
		stopGRPCServer(grpcServer)
		return errors.Join(err, shutdownErr, stopAndFlushFileStorage(stopFileStorage, fileStorageDone))
	case err := <-fileStorageDone:
		shutdownErr := httpServer.Shutdown(context.Background())
		stopGRPCServer(grpcServer)
		if errors.Is(shutdownErr, http.ErrServerClosed) {
			shutdownErr = nil
		}
		return errors.Join(err, shutdownErr)
	case <-ctx.Done():
		// Shutdown перестаёт принимать новые запросы и ждёт завершения уже
		// запущенных обработчиков. Только после этого фиксируем снимок в файл.
		shutdownErr := httpServer.Shutdown(context.Background())
		stopGRPCServer(grpcServer)
		storageErr := stopAndFlushFileStorage(stopFileStorage, fileStorageDone)
		return errors.Join(shutdownErr, storageErr)
	}
}

func stopGRPCServer(server *grpc.Server) {
	if server != nil {
		server.GracefulStop()
	}
}

func stopAndFlushFileStorage(cancel context.CancelFunc, done <-chan error) error {
	if cancel == nil {
		return nil
	}

	cancel()
	return <-done
}
