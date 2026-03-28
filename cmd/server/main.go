package main

import (
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
)

func main() {
	handler.Init()

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}

}
