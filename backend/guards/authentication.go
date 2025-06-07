package guards

import (
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func JWTGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utilities.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"No authorization token provided",
				nil,
			))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utilities.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"Invalid authorization header format",
				nil,
			))
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, schemas.NewAPIError(
					schemas.ErrInvalidCredentials,
					"Invalid token signing method",
					nil,
				)
			}
			return utilities.GetJWTSecret(), nil
		})

		if err != nil {
			utilities.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"Invalid or expired token",
				err,
			))
			return
		}

		if !token.Valid {
			utilities.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"Invalid token",
				nil,
			))
			return
		}

		next.ServeHTTP(w, r)
	})
}
