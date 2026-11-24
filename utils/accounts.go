package utils

import (
	"net/url"
	"strings"
)

// ExternalLink returns raw as a link to open, or "" when it is not one: only
// http and https URLs with a host qualify, and a missing scheme is taken as
// https, as the old sanitizeUrl did.
func ExternalLink(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return u.String()
}

// Initials is the badge letter: the platform's first character, upper-cased,
// or "?" when there is none.
func Initials(platform string) string {
	for _, r := range strings.TrimSpace(platform) {
		return strings.ToUpper(string(r))
	}
	return "?"
}
