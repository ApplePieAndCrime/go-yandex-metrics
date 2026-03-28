package main

import (
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/agent"
)

func main() {
	handler.Init()

	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		panic(err)
	}

}
