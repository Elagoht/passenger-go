package strength

import (
	"strings"
	"testing"
)

func TestScore(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", -2},
		{"short lower", "abc", -1},
		{"mixed", "Abcdefgh1!", 5},
		{"non-ascii counts runes only for length", strings.Repeat("ş", 21), 4},
	}
	for _, tt := range tests {
		if got := Score(tt.in); got != tt.want {
			t.Errorf("%s: Score(%q) = %d, want %d", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestLabel(t *testing.T) {
	tests := map[int]string{-2: "weak", 2: "weak", 3: "fair", 5: "fair", 6: "strong", 8: "strong"}
	for score, want := range tests {
		if got := Label(score); got != want {
			t.Errorf("Label(%d) = %q, want %q", score, got, want)
		}
	}
}
