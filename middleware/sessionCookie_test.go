package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"passenger-go/internal/session"
)

// fakePlugin stands in for the session plugin: it refreshes or clears the
// session cookie, and sets another cookie, before or after writing.
func fakePlugin(maxAge int, write bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := "e.payload.sig"
		if maxAge < 0 {
			value = ""
		}
		http.SetCookie(w, &http.Cookie{
			Name: session.CookieName, Value: value, Path: "/", MaxAge: maxAge,
			HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode,
		})
		http.SetCookie(w, &http.Cookie{Name: "other", Value: "kept", Path: "/"})
		if write {
			w.Write([]byte("{}"))
		}
	})
}

func sessionCookies(rec *httptest.ResponseRecorder) []*http.Cookie {
	var out []*http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == session.CookieName {
			out = append(out, c)
		}
	}
	return out
}

func TestSessionCookieQuietPaths(t *testing.T) {
	for _, tc := range []struct {
		path        string
		maxAge      int
		write       bool
		wantRefresh bool
	}{
		{"/", 900, true, true},
		{"/session/extend", 900, true, true},
		{"/session", 900, true, false},
		{"/session", 900, false, false}, // committed after the handler
		{"/static/app.css", 900, true, false},
	} {
		rec := httptest.NewRecorder()
		SessionCookie(fakePlugin(tc.maxAge, tc.write)).ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
		got := sessionCookies(rec)
		if (len(got) == 1) != tc.wantRefresh {
			t.Errorf("%s (write=%v): session cookies %v, want refreshed=%v", tc.path, tc.write, got, tc.wantRefresh)
		}
		found := false
		for _, c := range rec.Result().Cookies() {
			found = found || (c.Name == "other" && c.Value == "kept")
		}
		if !found {
			t.Errorf("%s: another cookie was dropped", tc.path)
		}
	}
}

// A cookie that clears the session is let through on the quiet paths: an
// expired session is still forgotten there.
func TestSessionCookieKeepsAClear(t *testing.T) {
	for _, path := range []string{"/session", "/static/app.css"} {
		rec := httptest.NewRecorder()
		SessionCookie(fakePlugin(-1, true)).ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		got := sessionCookies(rec)
		if len(got) != 1 || got[0].MaxAge >= 0 {
			t.Errorf("%s: session cookies %v, want the clearing one", path, got)
		}
	}
}

// The session cookie that is let through lasts as long as the browser session,
// with its other attributes kept.
func TestSessionCookieForTheBrowserSession(t *testing.T) {
	rec := httptest.NewRecorder()
	SessionCookie(fakePlugin(900, true)).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	var line string
	for _, l := range rec.Header().Values("Set-Cookie") {
		if strings.HasPrefix(l, session.CookieName+"=") {
			line = l
		}
	}
	for _, want := range []string{session.CookieName + "=e.payload.sig", "Path=/", "HttpOnly", "Secure", "SameSite=Strict"} {
		if !strings.Contains(line, want) {
			t.Errorf("Set-Cookie lacks %q: %s", want, line)
		}
	}
	for _, unwanted := range []string{"Max-Age", "Expires"} {
		if strings.Contains(line, unwanted) {
			t.Errorf("Set-Cookie has %s: %s", unwanted, line)
		}
	}
}
