// Package vault holds all cryptographic primitives for passenger-go.
package vault

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	algArgon2id = "argon2id"
	keyLen      = 32
	saltLen     = 16
)

// ErrWrongSecret is returned when a wrapped DEK cannot be opened.
var ErrWrongSecret = errors.New("wrong passphrase or recovery key")

// KDFParams are the Argon2id cost parameters, stored alongside each wrap.
type KDFParams struct {
	Alg string `json:"alg"`
	T   uint32 `json:"t"`
	M   uint32 `json:"m"`
	P   uint8  `json:"p"`
}

// DefaultKDF is the production Argon2id cost.
var DefaultKDF = KDFParams{"argon2id", 3, 65536, 4}

func (p KDFParams) derive(secret string, salt []byte) ([]byte, error) {
	if p.Alg != algArgon2id {
		return nil, fmt.Errorf("vault: unsupported kdf algorithm %q", p.Alg)
	}
	if p.T == 0 || p.M == 0 || p.P == 0 {
		return nil, errors.New("vault: invalid kdf parameters")
	}
	return argon2.IDKey([]byte(secret), salt, p.T, p.M, p.P, keyLen), nil
}
