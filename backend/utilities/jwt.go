package utilities

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var (
	jwtSecret []byte
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	jwtSecret = []byte(os.Getenv("JWT_SECRET"))

	if jwtSecret == nil {
		log.Fatal("JWT_SECRET is not set")
	}
}

func GetJWTSecret() []byte {
	return jwtSecret
}

func GenerateJWT(userId int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "passenger-go",
		"sub": userId,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(), // 1 hour expiration
	})

	return token.SignedString(jwtSecret)
}
