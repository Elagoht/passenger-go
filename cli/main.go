package main

import (
	"net/http"
	"os"

	"passenger-go/backend"
	"passenger-go/backend/middlewares"
	"passenger-go/backend/utilities/logger"

	"github.com/go-chi/chi"
)

func main() {
	log := logger.GetLogger()

	router := chi.NewRouter()

	router.Use(middlewares.SetAPIContentTypeJSON)

	router = backend.MountBackend(router)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
