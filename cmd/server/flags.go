package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

var flagRunAddress string
var flagInterval int64
var flagStoragePath string
var flagRestore bool

func parseFlags() {
	flag.StringVar(&flagRunAddress, "a", "localhost:8080", "адрес для старта сервера")
	flag.Int64Var(&flagInterval, "i", 300, "интервал времени в секундах, по истечении которого текущие показания сервера сохраняются на диск")
	flag.StringVar(&flagStoragePath, "f", "storage.json", "путь до файла, куда сохраняются текущие значения")
	flag.BoolVar(&flagRestore, "r", true, "определяет следует ли загружать ранее сохранённые значения из указанного файла при старте сервера")

	flag.Parse()

	if address := os.Getenv("ADDRESS"); address != "" {
		flagRunAddress = address
	}
	if interval := os.Getenv("STORE_INTERVAL"); interval != "" {
		flag, err := strconv.ParseInt(interval, 10, 64)
		if err != nil {
			fmt.Println("STORE_INTERVAL not valid number")
		}
		flagInterval = flag
	}
	if storagePath := os.Getenv("FILE_STORAGE_PATH"); storagePath != "" {
		flagStoragePath = storagePath
	}
	if isRestore := os.Getenv("RESTORE"); isRestore != "" {
		flag, err := strconv.ParseBool(isRestore)
		if err != nil {
			fmt.Println("RESTORE not valid bool")
		}
		flagRestore = flag
	}
}
