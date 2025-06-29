package auth

import (
	"net/http"
	"passenger-go/backend/services"
	"strings"
)

func PrivateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !CheckAuth(writer, request) {
			http.Redirect(writer, request, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func PublicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if CheckAuth(writer, request) {
			http.Redirect(writer, request, "/", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

var routesDoNotRequireInitialization = []string{
	"/register",
	"/recover",
	"/check",
	"/complete",
	"/static",
}

func InitializationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api") {
			next.ServeHTTP(writer, request)
			return
		}

		// Skip initialization check for public routes and static files
		for _, route := range routesDoNotRequireInitialization {
			if strings.HasPrefix(request.URL.Path, route) {
				next.ServeHTTP(writer, request)
				return
			}
		}

		// Check if application is initialized
		authService := services.NewAuthService()
		initialized, err := authService.Status()
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		if !initialized {
			http.Redirect(writer, request, "/register", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(writer, request)
	})
}
