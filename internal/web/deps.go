// Package web holds what the collage pages and actions share: the
// application's services, the brute-force backoff, the Argon2 concurrency cap
// and the pending setup.
package web

import (
	"sync"
	"time"

	fail2ban "github.com/Elagoht/collage-fail2ban"

	"passenger-go/internal/ratelimit"
	"passenger-go/internal/session"
	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

// Deps are the application's long-lived services, handed to every page and
// action constructor.
type Deps struct {
	Store    *store.Store
	Sessions *session.Manager
	Limiter  *ratelimit.Limiter
	// Bans is the elagoht/fail2ban plugin; Throttle strikes its SecretJail. Nil
	// in unit tests, where nothing is struck.
	Bans *fail2ban.Plugin
	KDF  vault.KDFParams
	Now  func() time.Time

	// SetupCode is asked for by /setup; Pending holds a setup awaiting its
	// recovery-key confirmation.
	SetupCode *SetupCode
	Pending   *PendingSetup

	// WrapMu is held by every request that changes the stored wraps, from
	// reading the vault row through storing the new wraps (and the session
	// changes that follow), so two of them never both start from the same row:
	// a double-submitted recovery cannot show a key that a parallel one
	// overwrote.
	WrapMu sync.Mutex
}
