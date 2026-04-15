package main

import (
	"flag"
	"os"
)

var flagRunAddress string

func parseFlags() {
	defaultAddress := "localhost:8080"
	if address := os.Getenv("ADDRESS"); address != "" {
		defaultAddress = address
	}

	flag.StringVar(&flagRunAddress, "a", defaultAddress, "address and port to run server")

	flag.Parse()
}
