// Package generator creates random passphrases using crypto/rand.
package generator

import (
	"crypto/rand"
	"math/big"
	"strings"
)

/**
 * Manipulate maps each character to similar-looking characters.
 * This means passphrase should be as powerful as possible while
 * still being easy to read and remember.
 */
var ManipulateMap = map[string][]string{
	"q": {"Q", "q"},
	"w": {"W", "m", "M", "w"},
	"e": {"E", "e"},
	"r": {"R", "r"},
	"t": {"T", "7", "t"},
	"y": {"Y", "h", "y"},
	"u": {"U", "u", "n"},
	"i": {"I", "1", "i"},
	"o": {"O", "0", "o"},
	"p": {"P", "p"},
	"a": {"A", "4", "@", "a"},
	"s": {"S", "$", "5", "s"},
	"d": {"D", "d"},
	"f": {"F", "f"},
	"g": {"G", "6", "9", "g"},
	"h": {"H", "y", "h"},
	"j": {"J", "j"},
	"k": {"K", "k"},
	"l": {"L", "l"},
	"z": {"Z", "2", "z"},
	"x": {"X", "x"},
	"c": {"C", "c"},
	"v": {"V", "v"},
	"b": {"B", "3", "8", "b"},
	"n": {"N", "n", "u"},
	"m": {"M", "W", "w", "m"},
	"$": {"S", "s", "5"},
	"@": {"A", "a"},
	"?": {"7"},
	"0": {"O", "o", "0"},
	"1": {"i", "1"},
	"2": {"Z", "z", "2"},
	"3": {"B", "3"},
	"4": {"A", "4"},
	"5": {"S", "s", "$"},
	"6": {"G", "6"},
	"7": {"7", "?", "T"},
	"8": {"B", "8"},
	"9": {"g", "9"},
}

const (
	Specials = "!@#$%^&*()_+-=[]{}|;:,.<>?/"
	Lowers   = "abcdefghijklmnopqrstuvyz"
	Uppers   = "ABCDEFGHIJKLMNOPQRSTUVYZ"
	Numbers  = "0123456789"
	Chars    = Lowers + Uppers + Numbers + Specials
)

func randIndex(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}

// Generate returns a random passphrase of the given length (clamped to
// [8, 4096]) containing at least two characters from each character set.
func Generate(length int) (string, error) {
	length = min(max(length, 8), 4096)

	passphrase := make([]byte, length)
	for i := range passphrase {
		idx, err := randIndex(len(Chars))
		if err != nil {
			return "", err
		}
		passphrase[i] = Chars[idx]
	}

	positions := make([]int, length)
	for i := range positions {
		positions[i] = i
	}
	for i := len(positions) - 1; i > 0; i-- {
		j, err := randIndex(i + 1)
		if err != nil {
			return "", err
		}
		positions[i], positions[j] = positions[j], positions[i]
	}

	sets := []string{Lowers, Uppers, Numbers, Specials}
	for n := range 8 {
		set := sets[n%4]
		idx, err := randIndex(len(set))
		if err != nil {
			return "", err
		}
		passphrase[positions[n]] = set[idx]
	}
	return string(passphrase), nil
}

// Alternate replaces each character with a random look-alike from
// ManipulateMap; unmapped characters are lowercased.
func Alternate(s string) (string, error) {
	var out strings.Builder
	for _, r := range s {
		lower := strings.ToLower(string(r))
		alts, ok := ManipulateMap[lower]
		if !ok {
			out.WriteString(lower)
			continue
		}
		idx, err := randIndex(len(alts))
		if err != nil {
			return "", err
		}
		out.WriteString(alts[idx])
	}
	return out.String(), nil
}
