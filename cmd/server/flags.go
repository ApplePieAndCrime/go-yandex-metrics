package main

import (
	"flag"
	"io"
	"os"

	configutil "github.com/ApplePieAndCrime/go-yandex-metrics/internal/config"
	"github.com/caarlos0/env/v11"
)

// FlagConfig содержит параметры запуска сервера.
type FlagConfig struct {
	RunAddress    string `env:"ADDRESS"`
	Interval      int64  `env:"STORE_INTERVAL"`
	StoragePath   string `env:"FILE_STORAGE_PATH"`
	IsRestore     bool   `env:"RESTORE"`
	DatabaseDsn   string `env:"DATABASE_DSN"`
	Key           string `env:"KEY"`
	AuditFile     string `env:"AUDIT_FILE"`
	AuditUrl      string `env:"AUDIT_URL"`
	CryptoKey     string `env:"CRYPTO_KEY"`
	TrustedSubnet string `env:"TRUSTED_SUBNET"`
	GRPCAddress   string `env:"GRPC_ADDRESS"`
	GRPCCertFile  string `env:"GRPC_CERT_FILE"`
	GRPCKeyFile   string `env:"GRPC_KEY_FILE"`
	Config        string
}

type fileConfig struct {
	Address       *string `json:"address"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"`
	StoreFile     *string `json:"store_file"`
	DatabaseDsn   *string `json:"database_dsn"`
	Key           *string `json:"key"`
	AuditFile     *string `json:"audit_file"`
	AuditURL      *string `json:"audit_url"`
	CryptoKey     *string `json:"crypto_key"`
	TrustedSubnet *string `json:"trusted_subnet"`
	GRPCAddress   *string `json:"grpc_address"`
	GRPCCertFile  *string `json:"grpc_cert_file"`
	GRPCKeyFile   *string `json:"grpc_key_file"`
}

func parseFlags() (*FlagConfig, error) {
	cfg := FlagConfig{
		RunAddress:    "localhost:8080",
		Interval:      300,
		StoragePath:   "storage.json",
		IsRestore:     true,
		DatabaseDsn:   "",
		Key:           "",
		AuditFile:     "",
		AuditUrl:      "",
		CryptoKey:     "",
		TrustedSubnet: "",
		GRPCAddress:   "",
		GRPCCertFile:  "",
		GRPCKeyFile:   "",
		Config:        "",
	}

	configPath, err := parseConfigPath(cfg, os.Args[1:], os.Getenv("CONFIG"))
	if err != nil {
		return nil, err
	}
	cfg.Config = configPath
	if cfg.Config != "" {
		if err := applyFileConfig(&cfg); err != nil {
			return nil, err
		}
	}

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	if storeFile, exists := os.LookupEnv("STORE_FILE"); exists {
		cfg.StoragePath = storeFile
	}

	registerFlags(flag.CommandLine, &cfg)
	if err := flag.CommandLine.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func parseConfigPath(cfg FlagConfig, args []string, envPath string) (string, error) {
	cfg.Config = envPath

	firstPass := flag.NewFlagSet("server-config", flag.ContinueOnError)
	firstPass.SetOutput(io.Discard)
	registerFlags(firstPass, &cfg)
	if err := firstPass.Parse(args); err != nil {
		return "", err
	}

	return cfg.Config, nil
}

func registerFlags(flagSet *flag.FlagSet, cfg *FlagConfig) {
	flagSet.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "адрес для старта сервера")
	flagSet.Int64Var(&cfg.Interval, "i", cfg.Interval, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск")
	flagSet.StringVar(&cfg.StoragePath, "f", cfg.StoragePath, "путь до файла, куда сохраняются текущие значения")
	flagSet.BoolVar(&cfg.IsRestore, "r", cfg.IsRestore, "определяет следует ли загружать ранее сохранённые значения из указанного файла при старте сервера")
	flagSet.StringVar(&cfg.DatabaseDsn, "d", cfg.DatabaseDsn, "строка подключения к базе данных")
	flagSet.StringVar(&cfg.Key, "k", cfg.Key, "ключ для авторизации")
	flagSet.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "путь к файлу, в который сохраняются логи аудита")
	flagSet.StringVar(&cfg.AuditUrl, "audit-url", cfg.AuditUrl, "полный URL, по которому отправляются логи аудита")
	flagSet.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "путь к файлу приватного ключа")
	flagSet.StringVar(&cfg.TrustedSubnet, "t", cfg.TrustedSubnet, "доверенная подсеть в формате CIDR")
	flagSet.StringVar(&cfg.GRPCAddress, "g", cfg.GRPCAddress, "адрес gRPC-сервера")
	flagSet.StringVar(&cfg.GRPCAddress, "grpc-address", cfg.GRPCAddress, "адрес gRPC-сервера")
	flagSet.StringVar(&cfg.GRPCCertFile, "grpc-cert", cfg.GRPCCertFile, "путь к TLS-сертификату gRPC-сервера")
	flagSet.StringVar(&cfg.GRPCKeyFile, "grpc-key", cfg.GRPCKeyFile, "путь к приватному ключу gRPC-сервера")
	flagSet.StringVar(&cfg.Config, "c", cfg.Config, "путь к JSON-файлу конфигурации")
	flagSet.StringVar(&cfg.Config, "config", cfg.Config, "путь к JSON-файлу конфигурации")
}

func applyFileConfig(cfg *FlagConfig) error {
	var fileCfg fileConfig
	if err := configutil.ReadJSON(cfg.Config, &fileCfg); err != nil {
		return err
	}

	if fileCfg.Address != nil {
		cfg.RunAddress = *fileCfg.Address
	}
	if fileCfg.Restore != nil {
		cfg.IsRestore = *fileCfg.Restore
	}
	if fileCfg.StoreInterval != nil {
		var err error
		cfg.Interval, err = configutil.DurationSeconds("store_interval", *fileCfg.StoreInterval)
		if err != nil {
			return err
		}
	}
	if fileCfg.StoreFile != nil {
		cfg.StoragePath = *fileCfg.StoreFile
	}
	if fileCfg.DatabaseDsn != nil {
		cfg.DatabaseDsn = *fileCfg.DatabaseDsn
	}
	if fileCfg.Key != nil {
		cfg.Key = *fileCfg.Key
	}
	if fileCfg.AuditFile != nil {
		cfg.AuditFile = *fileCfg.AuditFile
	}
	if fileCfg.AuditURL != nil {
		cfg.AuditUrl = *fileCfg.AuditURL
	}
	if fileCfg.CryptoKey != nil {
		cfg.CryptoKey = *fileCfg.CryptoKey
	}
	if fileCfg.TrustedSubnet != nil {
		cfg.TrustedSubnet = *fileCfg.TrustedSubnet
	}
	if fileCfg.GRPCAddress != nil {
		cfg.GRPCAddress = *fileCfg.GRPCAddress
	}
	if fileCfg.GRPCCertFile != nil {
		cfg.GRPCCertFile = *fileCfg.GRPCCertFile
	}
	if fileCfg.GRPCKeyFile != nil {
		cfg.GRPCKeyFile = *fileCfg.GRPCKeyFile
	}

	return nil
}
