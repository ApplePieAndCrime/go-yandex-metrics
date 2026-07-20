package main

import (
	"flag"

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
	}

	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	flag.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "адрес для старта сервера")
	flag.Int64Var(&cfg.Interval, "i", cfg.Interval, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск")
	flag.StringVar(&cfg.StoragePath, "f", cfg.StoragePath, "путь до файла, куда сохраняются текущие значения")
	flag.BoolVar(&cfg.IsRestore, "r", cfg.IsRestore, "определяет следует ли загружать ранее сохранённые значения из указанного файла при старте сервера")
	flag.StringVar(&cfg.DatabaseDsn, "d", cfg.DatabaseDsn, "строка подключения к базе данных")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "ключ для авторизации")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "путь к файлу, в который сохраняются логи аудита")
	flag.StringVar(&cfg.AuditUrl, "audit-url", cfg.AuditUrl, "полный URL, по которому отправляются логи аудита")

	flag.Parse()

	return &cfg, nil
}
