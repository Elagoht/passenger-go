package web

import (
	"errors"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
)

// SessionFrom returns the render's session, or nil when there is none.
func SessionFrom(rc *collage.RenderContext) *session.Session {
	if rc == nil || rc.Request == nil {
		return nil
	}
	s, ok := session.FromContext(rc.Request.Context())
	if !ok {
		return nil
	}
	return s
}

// ErrNoSession is returned by RequireSessionFrom when a render has no session,
// which a guarded page or action never sees.
var ErrNoSession = errors.New("web: no session on a guarded render")

// RequireSessionFrom is SessionFrom for handlers behind guards.SignedIn. It
// returns an independent copy of the session, DEK array included, so the
// caller can wipe its own copy after use without touching the request's.
func RequireSessionFrom(rc *collage.RenderContext) (*session.Session, error) {
	s := SessionFrom(rc)
	if s == nil {
		return nil, ErrNoSession
	}
	cp := *s // a struct copy: DEK is an array, so the key bytes are copied too
	return &cp, nil
}
