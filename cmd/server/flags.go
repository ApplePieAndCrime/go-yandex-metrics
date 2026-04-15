package main

import (
	"flag"
	"os"
)

var flagRunAddress string

func parseFlags() {
	defaultAddress := "localhost:8080"
	if serverPort := os.Getenv("SERVER_PORT"); serverPort != "" {
		defaultAddress = "localhost:" + serverPort
	}

	flag.StringVar(&flagRunAddress, "a", defaultAddress, "address and port to run server")

	flag.Parse()
}
