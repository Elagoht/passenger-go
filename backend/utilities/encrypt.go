package utilities

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

var (
	aesGCMSecret []byte
	aesOnce      sync.Once
)

func initAESGCM() error {
	secret := os.Getenv("AES_GCM_SECRET")
	if secret == "" {
		return errors.New("AES_GCM_SECRET environment variable is not set")
	}
	aesGCMSecret = []byte(secret)

	if len(aesGCMSecret) < 32 {
		return errors.New("AES_GCM_SECRET must be at least 32 bytes long")
	}
	return nil
}

func getAESGCMSecret() ([]byte, error) {
	var initErr error
	aesOnce.Do(func() {
		initErr = initAESGCM()
	})
	if initErr != nil {
		return nil, initErr
	}
	return aesGCMSecret, nil
}

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

func AESGCMEncrypt(data []byte) (string, error) {
	secret, err := getAESGCMSecret()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func AESGCMDecrypt(data string) ([]byte, error) {
	secret, err := getAESGCMSecret()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(secret)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(decodedData) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := decodedData[:nonceSize], decodedData[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
