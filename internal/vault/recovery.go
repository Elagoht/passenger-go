package vault

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
)

var recoveryEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewRecoveryKey returns 32 random bytes as base32, in dash-separated groups of 4.
func NewRecoveryKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("vault: generate recovery key: %w", err)
	}
	s := recoveryEncoding.EncodeToString(b)
	groups := make([]string, 0, len(s)/4+1)
	for i := 0; i < len(s); i += 4 {
		groups = append(groups, s[i:min(i+4, len(s))])
	}
	return strings.Join(groups, "-"), nil
}

// NormalizeRecoveryKey upper-cases s and strips spaces and dashes.
func NormalizeRecoveryKey(s string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.ToUpper(s))
}

// NewAccountID returns 16 random bytes, hex-encoded.
func NewAccountID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("vault: generate account id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
