package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

const (
	// AADPass binds a DEK wrapped under the passphrase.
	AADPass = "passenger/v1/dek/pass"
	// AADRecovery binds a DEK wrapped under the recovery key.
	AADRecovery = "passenger/v1/dek/rec"
)

// DEK is the 32-byte data encryption key.
type DEK [32]byte

// NewDEK returns a fresh random DEK.
func NewDEK() (DEK, error) {
	var d DEK
	if _, err := rand.Read(d[:]); err != nil {
		return DEK{}, fmt.Errorf("vault: generate dek: %w", err)
	}
	return d, nil
}

// Wipe zeroes the key material.
func (d *DEK) Wipe() {
	clear(d[:])
}

// Wrapped is a DEK encrypted under a key derived from a secret.
// Box is nonce || ciphertext.
type Wrapped struct {
	Salt []byte
	Box  []byte
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// seal encrypts plaintext and returns nonce || ciphertext.
func seal(key, plaintext []byte, aad string) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize(), gcm.NonceSize()+len(plaintext)+gcm.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("vault: generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, []byte(aad)), nil
}

var errShortBox = errors.New("vault: ciphertext too short")

// open decrypts nonce || ciphertext. GCM failures are returned as-is.
func open(key, box []byte, aad string) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(box) < ns+gcm.Overhead() {
		return nil, errShortBox
	}
	return gcm.Open(nil, box[:ns], box[ns:], []byte(aad))
}

// Wrap encrypts dek under a key derived from secret, with a new random salt.
func Wrap(secret string, dek DEK, params KDFParams, aad string) (Wrapped, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return Wrapped{}, fmt.Errorf("vault: generate salt: %w", err)
	}
	key, err := params.derive(secret, salt)
	if err != nil {
		return Wrapped{}, err
	}
	defer clear(key)
	box, err := seal(key, dek[:], aad)
	if err != nil {
		return Wrapped{}, err
	}
	return Wrapped{Salt: salt, Box: box}, nil
}

// Unwrap recovers the DEK. It returns ErrWrongSecret when authentication fails.
func Unwrap(secret string, w Wrapped, params KDFParams, aad string) (DEK, error) {
	key, err := params.derive(secret, w.Salt)
	if err != nil {
		return DEK{}, err
	}
	defer clear(key)
	pt, err := open(key, w.Box, aad)
	if err != nil {
		if errors.Is(err, errShortBox) {
			return DEK{}, err
		}
		return DEK{}, ErrWrongSecret
	}
	defer clear(pt)
	var d DEK
	if len(pt) != len(d) {
		return DEK{}, errors.New("vault: unexpected dek length")
	}
	copy(d[:], pt)
	return d, nil
}
