package auth

import (
	"net/http"

	"passenger-go/backend/utilities/jwtoken"

	"github.com/golang-jwt/jwt/v5"
)

func CheckAuth(
	writer http.ResponseWriter,
	request *http.Request,
) bool {
	secret := jwtoken.GetJWTSecret()

	token, err := request.Cookie("token")

	if err != nil {
		return false
	}

	parsedToken, err := jwt.Parse(token.Value, func(
		token *jwt.Token,
	) (any, error) {
		return secret, nil
	})

	if err != nil {
		return false
	}

	return parsedToken.Valid
}
