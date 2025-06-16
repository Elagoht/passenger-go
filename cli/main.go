package main

import (
	"crypto/tls"
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

	// Get certificate paths from environment variables
	certPath := os.Getenv("SSL_CERT_PATH")
	keyPath := os.Getenv("SSL_KEY_PATH")

	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP521,
			tls.CurveP384,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	// Create server
	server := &http.Server{
		Addr:         ":443",
		Handler:      router,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // Disable HTTP/2
	}

	// Start the server
	if certPath != "" && keyPath != "" {
		log.Printf("Starting HTTPS server on :443")
		if err := server.ListenAndServeTLS(certPath, keyPath); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	} else {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("Starting HTTP server on port %s", port)
		if err := http.ListenAndServe(":"+port, router); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}
}
