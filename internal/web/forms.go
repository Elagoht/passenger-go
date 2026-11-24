package web

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
)

// SafeNext returns raw when it is a local path — starting with "/" but not
// "//" or "/\", containing no backslash and no control character (browsers
// strip tab and newline, turning "/\t/evil.com" into "//evil.com"), before
// and after percent-decoding — and "/" otherwise, so a next parameter cannot
// redirect off-site.
func SafeNext(raw string) string {
	if !localPath(raw) {
		return "/"
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil || !localPath(decoded) {
		return "/"
	}
	return raw
}

func localPath(p string) bool {
	return strings.HasPrefix(p, "/") &&
		!strings.HasPrefix(p, "//") &&
		!strings.Contains(p, `\`) &&
		strings.IndexFunc(p, func(r rune) bool { return r < 0x20 || r == 0x7f }) < 0
}

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

// RequireSessionFrom is SessionFrom for handlers behind RequireSession. It
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

// MinPassphraseRunes is the shortest master passphrase accepted.
const MinPassphraseRunes = 12

// CheckNewPassphrase returns why a new master passphrase and its repeat are
// refused, or "" when they are accepted.
func CheckNewPassphrase(pass, confirm string) string {
	if utf8.RuneCountInString(pass) < MinPassphraseRunes {
		return "The passphrase must be at least 12 characters."
	}
	if pass != confirm {
		return "The passphrases do not match."
	}
	return ""
}
