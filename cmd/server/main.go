package main

import (
	"fmt"
	"net/http"

	handler "github.com/ApplePieAndCrime/go-yandex-metrics/internal/handler/server"
)

func main() {
	parseFlags()

	err := RunServer()
	if err != nil {
		panic(err)
	}
}

func RunServer() error {
	routes := handler.Init()
	fmt.Println("Server is running on address: " + flagRunAddress)
	err := http.ListenAndServe(flagRunAddress, routes)
	return err
}
