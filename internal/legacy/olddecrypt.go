// Package legacy migrates a database of the previous passenger-go release
// into the new vault format. It is the only package besides internal/vault
// that touches AES-GCM directly, because it has to read the old format.
package legacy

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	_ "modernc.org/sqlite"
)

const secretKey = "AES_GCM_SECRET"

// readSecret returns AES_GCM_SECRET from a dotenv file without touching the
// process environment. Lines are KEY=VALUE, optionally prefixed by "export ",
// with optional matching single or double quotes; lines starting with # are
// comments. The last assignment wins.
func readSecret(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}
	defer f.Close()
	var (
		value string
		found bool
	)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != secretKey {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		value, found = v, true
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read env file: %w", err)
	}
	if !found || value == "" {
		return nil, fmt.Errorf("%s is not set in %s", secretKey, path)
	}
	if len(value) != 32 {
		return nil, fmt.Errorf("%s must be 32 bytes long, got %d", secretKey, len(value))
	}
	return []byte(value), nil
}

// oldDecrypt reverses the old Encrypt and EncryptDeterministic: both produce
// base64-std(nonce || AES-256-GCM ciphertext) with a 12-byte nonce and no AAD.
func oldDecrypt(secret []byte, b64 string) (string, error) {
	block, err := aes.NewCipher(secret)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	ns := gcm.NonceSize()
	if len(data) < ns+gcm.Overhead() {
		return "", errors.New("ciphertext too short")
	}
	pt, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(pt) {
		clear(pt)
		return "", errors.New("decrypted text is not valid UTF-8")
	}
	s := string(pt)
	clear(pt)
	return s, nil
}

// checkOldSideFiles refuses to read an old database that still has a WAL or
// a rollback journal next to it: immutable mode would ignore their content.
func checkOldSideFiles(path string) error {
	for _, suffix := range []string{"-wal", "-journal"} {
		p := path + suffix
		if _, err := os.Lstat(p); err == nil {
			return fmt.Errorf("old database has a pending %s file (%s); stop the old app and open it once so it finishes writing, then retry", suffix, p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check %s: %w", p, err)
		}
	}
	return nil
}

// openOld opens the old database strictly read-only. immutable=1 keeps
// SQLite from creating lock, journal or WAL files next to it.
func openOld(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("old database: %w", err)
	}
	q := url.Values{}
	q.Set("mode", "ro")
	q.Set("immutable", "1")
	u := url.URL{Path: path}
	dsn := (&url.URL{Scheme: "file", Opaque: u.EscapedPath(), RawQuery: q.Encode()}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open old database: %w", err)
	}
	return db, nil
}
