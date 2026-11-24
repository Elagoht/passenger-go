package web

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

// SetupCode is the one-time code printed to the server log while no vault
// exists; /setup asks for it, so only someone who can read the log can claim
// a fresh install.
type SetupCode struct {
	code string
}

// NewSetupCode returns a random 8-digit code.
func NewSetupCode() (*SetupCode, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		return nil, fmt.Errorf("web: generate setup code: %w", err)
	}
	return &SetupCode{code: fmt.Sprintf("%08d", n.Int64())}, nil
}

// Check reports, in constant time, whether s (surrounding space ignored) is
// the code.
func (c *SetupCode) Check(s string) bool {
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(s)), []byte(c.code)) == 1
}

// String returns the code, for the startup log.
func (c *SetupCode) String() string { return c.code }

// PendingCookieName is the cookie naming the setup waiting for its recovery
// key to be confirmed.
const PendingCookieName = "passenger_setup"

// pendingLifetime is how long a setup may wait for its confirmation.
const pendingLifetime = 15 * time.Minute

// PendingSetup holds the one setup that passed /setup and waits for the
// recovery key to be typed back on /setup/confirm. Nothing is written to disk
// until then. It holds at most one setup: a new Put replaces the previous one.
type PendingSetup struct {
	mu       sync.Mutex
	now      func() time.Time
	token    string
	dek      vault.DEK
	row      store.VaultRow
	recovery string
	created  time.Time
}

// NewPendingSetup returns an empty PendingSetup reading time from now.
func NewPendingSetup(now func() time.Time) *PendingSetup {
	return &PendingSetup{now: now}
}

// Put stores a setup, replacing (and wiping) any previous one, and returns the
// token its cookie carries. dek is copied.
func (p *PendingSetup) Put(dek vault.DEK, row store.VaultRow, recovery string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("web: generate setup token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clearLocked()
	p.token, p.dek, p.row, p.recovery, p.created = token, dek, row, recovery, p.now()
	return token, nil
}

// Get returns the setup token names when it is the current one and younger
// than 15 minutes. The DEK is a copy the caller wipes.
func (p *PendingSetup) Get(token string) (row store.VaultRow, recovery string, dek vault.DEK, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(p.token)) != 1 {
		return store.VaultRow{}, "", vault.DEK{}, false
	}
	if p.now().Sub(p.created) >= pendingLifetime {
		p.clearLocked()
		return store.VaultRow{}, "", vault.DEK{}, false
	}
	return p.row, p.recovery, p.dek, true
}

// Clear drops the pending setup and wipes its DEK.
func (p *PendingSetup) Clear() {
	p.mu.Lock()
	p.clearLocked()
	p.mu.Unlock()
}

func (p *PendingSetup) clearLocked() {
	p.dek.Wipe()
	p.token, p.row, p.recovery, p.created = "", store.VaultRow{}, "", time.Time{}
}

// PendingCookie is the cookie carrying a pending setup's token, for the
// browser's session (validity is server-side): HttpOnly, SameSite=Strict,
// Path=/, Secure over TLS. It is not the session cookie, which the
// elagoht/session plugin writes: a pending setup is not signed in.
func PendingCookie(r *http.Request, token string) *http.Cookie {
	return &http.Cookie{
		Name:     PendingCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   collage.IsHTTPS(r),
	}
}

// ClearPendingCookie is the pending-setup cookie that tells the browser to
// drop it.
func ClearPendingCookie(r *http.Request) *http.Cookie {
	c := PendingCookie(r, "")
	c.MaxAge = -1
	return c
}

// PendingToken returns the pending-setup token r carries, or "".
func PendingToken(r *http.Request) string {
	c, err := r.Cookie(PendingCookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// HasPendingSetup reports whether r carries the token of a live pending
// setup.
func (d *Deps) HasPendingSetup(r *http.Request) bool {
	_, _, dek, ok := d.Pending.Get(PendingToken(r))
	dek.Wipe()
	return ok
}
