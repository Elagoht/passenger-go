package vault

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

var fast = KDFParams{"argon2id", 1, 1024, 1}

func mustDEK(t *testing.T) DEK {
	t.Helper()
	d, err := NewDEK()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestWrapUnwrapRoundTrip(t *testing.T) {
	dek := mustDEK(t)
	w1, err := Wrap("pw", dek, fast, AADPass)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Unwrap("pw", w1, fast, AADPass)
	if err != nil || got != dek {
		t.Fatalf("round trip failed: %v", err)
	}
	w2, _ := Wrap("pw", dek, fast, AADPass)
	if bytes.Equal(w1.Salt, w2.Salt) || bytes.Equal(w1.Box, w2.Box) {
		t.Fatal("two wraps must differ")
	}
}

func TestUnwrapFailures(t *testing.T) {
	dek := mustDEK(t)
	w, _ := Wrap("pw", dek, fast, AADPass)
	if _, err := Unwrap("nope", w, fast, AADPass); !errors.Is(err, ErrWrongSecret) {
		t.Fatalf("wrong secret: %v", err)
	}
	if _, err := Unwrap("pw", w, fast, AADRecovery); !errors.Is(err, ErrWrongSecret) {
		t.Fatalf("wrong aad: %v", err)
	}
	bad := fast
	bad.Alg = "scrypt"
	if _, err := Unwrap("pw", w, bad, AADPass); err == nil {
		t.Fatal("unsupported alg must error")
	}
	if _, err := Unwrap("pw", Wrapped{Salt: w.Salt, Box: []byte{1}}, fast, AADPass); err == nil {
		t.Fatal("short box must error")
	}
}

func TestWipe(t *testing.T) {
	d := mustDEK(t)
	d.Wipe()
	if d != (DEK{}) {
		t.Fatal("not wiped")
	}
}

func TestSealOpen(t *testing.T) {
	dek := mustDEK(t)
	r := Record{"GitHub", "me", "s3cret", "https://github.com", "n"}
	blob, err := Seal(dek, "id1", r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Open(dek, "id1", blob)
	if err != nil || got != r {
		t.Fatalf("open: %v %+v", err, got)
	}
	if _, err := Open(dek, "id2", blob); err == nil {
		t.Fatal("different id must fail")
	}
	blob[len(blob)-1] ^= 1
	if _, err := Open(dek, "id1", blob); err == nil {
		t.Fatal("tampered blob must fail")
	}
}

func TestLookup(t *testing.T) {
	dek := mustDEK(t)
	if !bytes.Equal(Lookup(dek, "GitHub ", "a"), Lookup(dek, "github", "a")) {
		t.Fatal("platform must be case/space-insensitive")
	}
	if bytes.Equal(Lookup(dek, "github", "a"), Lookup(dek, "github", "A")) {
		t.Fatal("identifier is case-sensitive")
	}
	if bytes.Equal(Lookup(dek, "github", "a"), Lookup(mustDEK(t), "github", "a")) {
		t.Fatal("must differ across DEKs")
	}
	if !bytes.Equal(Lookup(dek, "İnstagram", "x"), Lookup(dek, "İnstagram", "x")) {
		t.Fatal("must be stable")
	}
}

func TestRecoveryKey(t *testing.T) {
	k, err := NewRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.ReplaceAll(k, "-", "")); n != 52 {
		t.Fatalf("want 52 chars, got %d", n)
	}
	if NormalizeRecoveryKey("abcd-efgh ijkl") != "ABCDEFGHIJKL" {
		t.Fatal("normalize")
	}
	dek := mustDEK(t)
	w, err := Wrap(NormalizeRecoveryKey(k), dek, fast, AADRecovery)
	if err != nil {
		t.Fatal(err)
	}
	variant := strings.ToLower(strings.ReplaceAll(k, "-", " "))
	got, err := Unwrap(NormalizeRecoveryKey(variant), w, fast, AADRecovery)
	if err != nil || got != dek {
		t.Fatalf("variant unwrap: %v", err)
	}
}

func TestNormalizeRecoveryKeyStripsAllWhitespaceAndDashes(t *testing.T) {
	for in, want := range map[string]string{
		"abcd\tefgh\nijkl":         "ABCDEFGHIJKL",
		"  abcd-efgh\r\n":          "ABCDEFGH",
		"abcd\u00a0efgh\u2003ijkl": "ABCDEFGHIJKL", // no-break and em space
		"abcd\u2013efgh\u2014ijkl": "ABCDEFGHIJKL", // en and em dash
		"ABCD-EFGH-IJKL":           "ABCDEFGHIJKL",
	} {
		if got := NormalizeRecoveryKey(in); got != want {
			t.Errorf("NormalizeRecoveryKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAccountID(t *testing.T) {
	a, _ := NewAccountID()
	b, _ := NewAccountID()
	if len(a) != 32 || a == b {
		t.Fatalf("bad ids %q %q", a, b)
	}
}
