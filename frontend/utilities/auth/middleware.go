package auth

import (
	"net/http"
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
