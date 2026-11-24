package web

import (
	"net/url"
	"strings"

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
