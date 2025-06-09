package guards

import (
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/api_error"
	"passenger-go/backend/utilities/jwtoken"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func JWTGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api_error.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"No authorization token provided",
				nil,
			))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			api_error.HandleAPIError(w, schemas.NewAPIError(
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
			return jwtoken.GetJWTSecret(), nil
		})

		if err != nil {
			api_error.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"Invalid or expired token",
				err,
			))
			return
		}

		if !token.Valid {
			api_error.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"Invalid token",
				nil,
			))
			return
		}

		next.ServeHTTP(w, r)
	})
}
