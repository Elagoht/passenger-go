package auth

import (
	"net/http"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !CheckAuth(writer, request) {
			http.Redirect(writer, request, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(writer, request)
	})
}
