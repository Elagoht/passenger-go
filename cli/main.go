package main

import (
	"log"
	"net/http"
	"os"

	"passenger-go/backend"
	"passenger-go/backend/middlewares"

	"github.com/go-chi/chi"
)

func main() {
	// Initialize logger
	logger := log.New(os.Stdout, "[Passenger] ", log.LstdFlags|log.Lshortfile)

	router := chi.NewRouter()

	router.Use(middlewares.SetAPIContentTypeJSON)

	router = backend.MountBackend(router)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}
