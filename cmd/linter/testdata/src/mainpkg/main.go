package main

import (
	"log"
	"os"
)

func main() {
	log.Fatal("allowed")
	os.Exit(0)
	deferred := func() {
		os.Exit(2)
	}
	_ = deferred
}

func helper() {
	log.Fatal("forbidden")
	os.Exit(1)
}
