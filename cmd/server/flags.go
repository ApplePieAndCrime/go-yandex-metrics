package main

import (
	"flag"
	"os"

	configutil "github.com/ApplePieAndCrime/go-yandex-metrics/internal/config"
	"github.com/caarlos0/env/v11"
)

// FlagConfig содержит параметры запуска сервера.
type FlagConfig struct {
	RunAddress  string `env:"ADDRESS"`
	Interval    int64  `env:"STORE_INTERVAL"`
	StoragePath string `env:"FILE_STORAGE_PATH"`
	IsRestore   bool   `env:"RESTORE"`
	DatabaseDsn string `env:"DATABASE_DSN"`
	Key         string `env:"KEY"`
	AuditFile   string `env:"AUDIT_FILE"`
	AuditUrl    string `env:"AUDIT_URL"`
	CryptoKey   string `env:"CRYPTO_KEY"`
	Config      string
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
}

func parseFlags() (*FlagConfig, error) {
	cfg := FlagConfig{
		RunAddress:  "localhost:8080",
		Interval:    300,
		StoragePath: "storage.json",
		IsRestore:   true,
		DatabaseDsn: "",
		Key:         "",
		AuditFile:   "",
		AuditUrl:    "",
		CryptoKey:   "",
		Config:      "",
	}

	cfg.Config = configutil.FilePath(os.Args[1:], os.Getenv("CONFIG"))
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

	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "адрес для старта сервера")
	flag.Int64Var(&cfg.Interval, "i", cfg.Interval, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск")
	flag.StringVar(&cfg.StoragePath, "f", cfg.StoragePath, "путь до файла, куда сохраняются текущие значения")
	flag.BoolVar(&cfg.IsRestore, "r", cfg.IsRestore, "определяет следует ли загружать ранее сохранённые значения из указанного файла при старте сервера")
	flag.StringVar(&cfg.DatabaseDsn, "d", cfg.DatabaseDsn, "строка подключения к базе данных")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "ключ для авторизации")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "путь к файлу, в который сохраняются логи аудита")
	flag.StringVar(&cfg.AuditUrl, "audit-url", cfg.AuditUrl, "полный URL, по которому отправляются логи аудита")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "путь к файлу приватного ключа")
	flag.StringVar(&cfg.Config, "c", cfg.Config, "путь к JSON-файлу конфигурации")
	flag.StringVar(&cfg.Config, "config", cfg.Config, "путь к JSON-файлу конфигурации")

	flag.Parse()

	return &cfg, nil
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

	return nil
}
