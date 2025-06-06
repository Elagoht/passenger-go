package utilities

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

func Encrypt(passphrase string) (string, error) {
	hash := argon2.IDKey(
		[]byte(passphrase),
		[]byte(os.Getenv("SALT")),
		1,
		64*1024,
		4,
		32,
	)
	return base64.StdEncoding.EncodeToString(hash), nil
}

func GenerateRecoveryKey(passphrase string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	derivedKey := pbkdf2.Key([]byte(passphrase), salt, 100000, 32, sha256.New)

	combined := append(salt, derivedKey...)
	recoveryKey := base64.URLEncoding.EncodeToString(combined)

	return recoveryKey, nil
}
