package url

import (
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/publicsuffix"
)

func ConvertURLToPlatformName(givenURL string) string {
	parsedURL, err := url.Parse(givenURL)
	if err != nil || parsedURL.Host == "" {
		return "Unknown"
	}

	host := parsedURL.Host
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return "Unknown"
	}

	parts := strings.Split(eTLDPlusOne, ".")
	if len(parts) == 0 {
		return "Unknown"
	}

	platform := parts[0]
	return capitalize(platform)
}

func capitalize(word string) string {
	if word == "" {
		return ""
	}
	runes := []rune(word)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
