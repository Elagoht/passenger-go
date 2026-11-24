package utils

import "testing"

// The cases of the old web.SafeNext test, all kept: the off-site ones go to
// the fallback.
func TestSafeRedirect(t *testing.T) {
	for raw, want := range map[string]string{
		"/accounts/1?x=1":  "/accounts/1?x=1",
		"/":                "/",
		"//evil.com":       "/",
		"/\\evil.com":      "/",
		"https://evil.com": "/",
		"/%2F%2Fevil.com":  "/",
		"/%5Cevil.com":     "/",
		"/a\\b":            "/",
		"":                 "/",
		"accounts":         "/",
		"/%zz":             "/",
		"/\t/evil.com":     "/",
		"/%09/evil.com":    "/",
		"/%0A/evil.com":    "/",
	} {
		if got := SafeRedirect(raw, "/"); got != want {
			t.Errorf("SafeRedirect(%q) = %q, want %q", raw, got, want)
		}
	}
	if got := SafeRedirect("//evil.com", "/home"); got != "/home" {
		t.Errorf("fallback = %q, want /home", got)
	}
	if got := SafeRedirect("/%2F%2Fevil.com", "/home"); got != "/home" {
		t.Errorf("decoded fallback = %q, want /home", got)
	}
}
