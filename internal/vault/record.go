package vault

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"strings"
)

const (
	aadAccountPrefix = "passenger/v1/account/"
	indexInfo        = "passenger/v1/index"
)

// Record is the plaintext content of one stored account.
type Record struct {
	Platform   string `json:"platform"`
	Identifier string `json:"identifier"`
	Passphrase string `json:"passphrase"`
	URL        string `json:"url"`
	Notes      string `json:"notes"`
}

// Seal encrypts r, binding the ciphertext to the account id.
func Seal(dek DEK, id string, r Record) ([]byte, error) {
	pt, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	defer clear(pt)
	return seal(dek[:], pt, aadAccountPrefix+id)
}

// Open decrypts a blob produced by Seal for the same id.
func Open(dek DEK, id string, blob []byte) (Record, error) {
	pt, err := open(dek[:], blob, aadAccountPrefix+id)
	if err != nil {
		return Record{}, err
	}
	defer clear(pt)
	var r Record
	if err := json.Unmarshal(pt, &r); err != nil {
		return Record{}, err
	}
	return r, nil
}

// Lookup returns a deterministic blind index for (platform, identifier).
func Lookup(dek DEK, platform, identifier string) []byte {
	// Key length 32 is far below the HKDF limit, so this cannot fail.
	key, err := hkdf.Key(sha256.New, dek[:], nil, indexInfo, 32)
	if err != nil {
		panic("vault: hkdf: " + err.Error())
	}
	defer clear(key)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(strings.ToLower(strings.TrimSpace(platform)) + "\x00" + identifier))
	return mac.Sum(nil)
}
