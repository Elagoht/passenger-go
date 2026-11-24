// Package transfer reads and writes the CSV files accounts move in and out
// with: Firefox and Chromium password exports and Passenger's own format.
package transfer

import (
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/publicsuffix"
)

const unknownPlatform = "Unknown"

// PlatformFromURL names the site raw points at: the first label of its
// registrable domain, capitalised ("https://accounts.google.com/x" is
// "Google"). It is "Unknown" when raw has no host or no registrable domain.
func PlatformFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return unknownPlatform
	}
	domain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
	if err != nil {
		return unknownPlatform
	}
	label, _, _ := strings.Cut(domain, ".")
	runes := []rune(label)
	if len(runes) == 0 {
		return unknownPlatform
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
