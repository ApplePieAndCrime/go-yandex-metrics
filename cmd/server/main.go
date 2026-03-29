package main

import (
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
)

func main() {
	routes := handler.Init()

	err := http.ListenAndServe(":8080", routes)
	if err != nil {
		panic(err)
	}

}
