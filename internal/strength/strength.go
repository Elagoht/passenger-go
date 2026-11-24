// Package strength scores passphrases.
package strength

import (
	"strings"
	"unicode/utf8"
)

var characterSets = []string{
	"abcdefghijklmnopqrstuvwxyz",
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	"0123456789",
	"!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~",
}

// Score returns the length score plus one point per character set present.
func Score(passphrase string) int {
	return lengthScore(utf8.RuneCountInString(passphrase)) + characterSetScore(passphrase)
}

// Label maps a score to "weak", "fair" or "strong".
func Label(score int) string {
	switch {
	case score <= 2:
		return "weak"
	case score <= 5:
		return "fair"
	default:
		return "strong"
	}
}

func lengthScore(length int) int {
	switch {
	case length <= 5:
		return -2
	case length <= 7:
		return 0
	case length <= 12:
		return 1
	case length <= 16:
		return 2
	case length <= 20:
		return 3
	default:
		return 4
	}
}

func characterSetScore(passphrase string) int {
	score := 0
	for _, set := range characterSets {
		if strings.ContainsAny(passphrase, set) {
			score++
		}
	}
	return score
}
