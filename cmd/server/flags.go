package main

import "flag"

var flagRunAddress string

func parseFlags() {
	flag.StringVar(&flagRunAddress, "a", "http://localhost:8080", "address and port to run server")

	flag.Parse()
}
