package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"passenger-go/backend/utilities/logger"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/pbkdf2"
)

var aesGCMSecret = []byte{}

func init() {
	godotenv.Load()
	log := logger.GetLogger()
	aesGCMSecret = []byte(os.Getenv("AES_GCM_SECRET"))
	if string(aesGCMSecret) == "" {
		log.Fatal("AES_GCM_SECRET environment variable is not set")
	}

	if len(aesGCMSecret) != 32 {
		log.Fatal("AES_GCM_SECRET must be 32 bytes long")
	}
}

// HashPassword creates a secure one-way hash of the password using Argon2
func HashPassword(password string) (string, error) {
	hash := argon2.IDKey(
		[]byte(password),
		[]byte(os.Getenv("SALT")),
		1,
		64*1024,
		4,
		32,
	)
	return base64.StdEncoding.EncodeToString(hash), nil
}

// VerifyPassword checks if the provided password matches the hash
func VerifyPassword(password, hashedPassword string) (bool, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	return hash == hashedPassword, nil
}

// Encrypt encrypts data using AES-GCM and returns a base64 encoded string
func Encrypt(data string) (string, error) {
	return aesGCMEncrypt([]byte(data))
}

// Decrypt decrypts a base64 encoded encrypted string
func Decrypt(encryptedData string) (string, error) {
	decrypted, err := aesGCMDecrypt(encryptedData)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// EncryptDeterministic encrypts data deterministically for database uniqueness
// WARNING: This is less secure than random encryption but needed for database constraints
func EncryptDeterministic(data string) (string, error) {
	return aesGCMEncryptDeterministic([]byte(data))
}

// DecryptDeterministic decrypts deterministically encrypted data
func DecryptDeterministic(encryptedData string) (string, error) {
	decrypted, err := aesGCMDecryptDeterministic(encryptedData)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
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

func aesGCMEncrypt(data []byte) (string, error) {
	block, err := aes.NewCipher(aesGCMSecret)
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

func aesGCMDecrypt(data string) ([]byte, error) {
	block, err := aes.NewCipher(aesGCMSecret)
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

// AESGCMEncryptDeterministic encrypts data with a deterministic nonce derived from the data
// This always produces the same ciphertext for the same input, suitable for database uniqueness
func aesGCMEncryptDeterministic(data []byte) (string, error) {
	block, err := aes.NewCipher(aesGCMSecret)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create deterministic nonce by hashing the data + secret
	hash := sha256.Sum256(append(data, aesGCMSecret...))
	nonce := hash[:gcm.NonceSize()] // Use first 12 bytes of hash as nonce

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESGCMDecryptDeterministic decrypts deterministically encrypted data
func aesGCMDecryptDeterministic(data string) ([]byte, error) {
	block, err := aes.NewCipher(aesGCMSecret)
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
