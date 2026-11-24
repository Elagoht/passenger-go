package generator

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestGenerateClamps(t *testing.T) {
	for in, want := range map[int]int{3: 8, 10000: 4096, 16: 16} {
		got, err := Generate(in)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != want {
			t.Errorf("Generate(%d) length = %d, want %d", in, len(got), want)
		}
	}
}

func TestGenerateContainsTwoOfEachSet(t *testing.T) {
	for range 200 {
		got, err := Generate(16)
		if err != nil {
			t.Fatal(err)
		}
		for name, set := range map[string]string{"lower": Lowers, "upper": Uppers, "number": Numbers, "special": Specials} {
			count := 0
			for _, r := range got {
				if strings.ContainsRune(set, r) {
					count++
				}
			}
			if count < 2 {
				t.Fatalf("%q has %d %s chars, want >= 2", got, count, name)
			}
		}
	}
}

func TestAlternate(t *testing.T) {
	in := "password"
	for range 100 {
		got, err := Alternate(in)
		if err != nil {
			t.Fatal(err)
		}
		if utf8.RuneCountInString(got) != utf8.RuneCountInString(in) {
			t.Fatalf("Alternate(%q) = %q, rune count differs", in, got)
		}
		for i, r := range []rune(got) {
			allowed := ManipulateMap[string(in[i])]
			if !contains(allowed, string(r)) {
				t.Fatalf("Alternate(%q) = %q: %q not in %v", in, got, string(r), allowed)
			}
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
