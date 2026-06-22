package main

import (
	"log"
	"net/http"

	"club/services/api/internal/app"
)

func main() {
	config := app.LoadConfig()

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: app.NewRouterWithConfig(config),
	}

	log.Printf("club api listening on http://localhost:%s", config.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
