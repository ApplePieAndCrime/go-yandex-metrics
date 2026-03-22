package main

import (
	"net/http"

	"github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler"
)

func main() {
	handler.Init()

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}

}
