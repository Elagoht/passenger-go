// Package web holds what the collage pages and actions share: dependencies,
// guards, security headers and form helpers.
package web

import (
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
}
