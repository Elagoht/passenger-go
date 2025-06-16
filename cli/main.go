package main

import (
	"net/http"
	"os"

	"passenger-go/backend"
	"passenger-go/backend/middlewares"
	"passenger-go/backend/utilities/logger"
	"passenger-go/frontend"

	"github.com/go-chi/chi"
)

func main() {
	log := logger.GetLogger()

	router := chi.NewRouter()

	// Initialize frontend controller
	frontendController, err := frontend.NewFrontendController()
	if err != nil {
		log.Fatalf("Failed to initialize frontend controller: %v", err)
	}

	// Mount frontend routes first
	frontendController.MountFrontendRouter(router)

	// Mount API routes with JSON content type middleware
	apiRouter := chi.NewRouter()
	apiRouter.Use(middlewares.SetAPIContentTypeJSON)
	apiRouter = backend.MountBackend(apiRouter)
	router.Mount("/", apiRouter)

	// Create server
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalf("PORT environment variable is not set")
	}
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start the server
	log.Printf("Starting server on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
