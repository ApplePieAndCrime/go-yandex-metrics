package main

import (
	"log"
	"os"
)

func main() {
	log.Fatal("allowed")
	os.Exit(0)
	deferred := func() {
		os.Exit(2) // want "вызов os.Exit разрешён только в функции main пакета main"
	}
	_ = deferred
}

func helper() {
	log.Fatal("forbidden") // want "вызов log.Fatal разрешён только в функции main пакета main"
	os.Exit(1)             // want "вызов os.Exit разрешён только в функции main пакета main"
}
