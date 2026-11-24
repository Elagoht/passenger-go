package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"passenger-go/internal/session"
)

// cookieWriter stands in for the session plugin: it refreshes or clears the
// session cookie, and sets another cookie, before or after writing.
func cookieWriter(maxAge int, write bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := "e.payload.sig"
		if maxAge < 0 {
			value = ""
		}
		http.SetCookie(w, &http.Cookie{Name: session.CookieName, Value: value, Path: "/", MaxAge: maxAge})
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

func TestQuietSession(t *testing.T) {
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
		QuietSession(cookieWriter(tc.maxAge, tc.write)).ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
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
func TestQuietSessionKeepsAClear(t *testing.T) {
	for _, path := range []string{"/session", "/static/app.css"} {
		rec := httptest.NewRecorder()
		QuietSession(cookieWriter(-1, true)).ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		got := sessionCookies(rec)
		if len(got) != 1 || got[0].MaxAge >= 0 {
			t.Errorf("%s: session cookies %v, want the clearing one", path, got)
		}
	}
}
