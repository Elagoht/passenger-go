package middlewares

import (
	"net/http"
	"strings"
)

func SetAPIContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			writer.Header().Set("Content-Type", "application/json")
		}
		next.ServeHTTP(writer, request)
	})
}
