// Package web holds what the collage pages and actions share: dependencies,
// guards, security headers and form helpers.
package web

import (
	"sync"
	"time"

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
	KDF      vault.KDFParams
	Now      func() time.Time

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
