package guards

import (
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/api_error"
	"passenger-go/backend/utilities/jwtoken"

	"github.com/golang-jwt/jwt/v5"
)

func JWTGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCookie, err := r.Cookie("token")
		if err != nil {
			api_error.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"No authorization token provided",
				nil,
			))
			return
		}

		if authCookie.Value == "" {
			api_error.HandleAPIError(w, schemas.NewAPIError(
				schemas.ErrInvalidCredentials,
				"No authorization token provided",
				nil,
			))
			return
		}

		token, err := jwt.Parse(authCookie.Value, func(token *jwt.Token) (any, error) {
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
