package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/caarlos0/env/v11"
)

type FlagConfig struct {
	RunAddress  string `env:"ADDRESS"`
	Interval    int64  `env:"STORE_INTERVAL"`
	StoragePath string `env:"FILE_STORAGE_PATH"`
	IsRestore   bool   `env:"RESTORE"`
	DatabaseDsn string `env:"DATABASE_DSN"`
}

func parseFlags() (*FlagConfig, error) {
	var cfg FlagConfig

	err := env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	log.Println("SERVER CONFIG ", cfg)

	if cfg.RunAddress != "" {
		cfg.RunAddress = cfg.RunAddress
	} else {
		flag.StringVar(&cfg.RunAddress, "a", "localhost:8080", "адрес для старта сервера")
	}

	if cfg.Interval != 0 {
		cfg.Interval = cfg.Interval
	} else {
		flag.Int64Var(&cfg.Interval, "i", 300, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск")
	}

	if cfg.StoragePath != "" {
		cfg.StoragePath = cfg.StoragePath
	} else {
		flag.StringVar(&cfg.StoragePath, "f", "storage.json", "путь до файла, куда сохраняются текущие значения")
	}

	if value, ok := os.LookupEnv("RESTORE"); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		} else {
			cfg.IsRestore = parsed
		}
	} else {
		flag.BoolVar(&cfg.IsRestore, "r", true, "определяет следует ли загружать ранее сохранённые значения из указанного файла при старте сервера")
	}

	flag.Parse()

	return &cfg, nil
}
