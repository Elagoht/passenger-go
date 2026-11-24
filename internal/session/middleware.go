package session

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey int

const (
	sessionKey ctxKey = iota
	hadCookieKey
)

func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey, s)
}

func FromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey).(*Session)
	return s, ok && s != nil
}

// HadCookie reports whether the request carried a session cookie that turned
// out invalid or expired.
func HadCookie(ctx context.Context) bool {
	v, _ := ctx.Value(hadCookieKey).(bool)
	return v
}

// Middleware resolves the session cookie. /session only peeks (so polling does
// not keep the session alive); /static/ is skipped entirely (fetching an asset
// is not activity); every other path extends the session. The request's copy
// of the session, DEK included, is wiped once the request is served.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}
		c, err := r.Cookie(CookieName)
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		var s *Session
		var ok bool
		if r.URL.Path == "/session" {
			s, ok = m.Peek(c.Value)
		} else {
			s, ok = m.Touch(c.Value)
		}
		ctx := r.Context()
		if ok {
			ctx = WithSession(ctx, s)
			defer s.DEK.Wipe()
		} else {
			ctx = context.WithValue(ctx, hadCookieKey, true)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
