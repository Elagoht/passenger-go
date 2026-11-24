// Package utils holds the helpers the actions and fragments share.
package utils

import (
	"net/url"

	"github.com/Elagoht/collage/pkg/collage"
)

// SafeRedirect is collage.SafeRedirect(next, fallback), with next also checked
// once percent-decoded: collage looks at the bytes as they are, so it lets
// "/%2F%2Fevil.com" or "/%09/evil.com" through. A browser does not decode a
// Location before following it, so those stay on this site, but a path that
// only decodes to a local one is no next this site wrote, and anything that
// unescapes it on the way (a proxy, a later redirect) would turn it into one
// off-site. A next that does not decode is refused too.
func SafeRedirect(next, fallback string) string {
	safe := collage.SafeRedirect(next, fallback)
	if safe != next {
		return safe
	}
	decoded, err := url.PathUnescape(next)
	if err != nil || collage.SafeRedirect(decoded, "") != decoded {
		return collage.SafeRedirect("", fallback)
	}
	return next
}
