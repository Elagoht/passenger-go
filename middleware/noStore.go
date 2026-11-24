package middleware

import (
	"net/http"
	"strings"
)

// NoStore sends Cache-Control: no-store on every response but the static
// files under /static/ (they hold no secrets and may be cached): a vault page
// must not stay in any cache. elagoht/secure sends the other security headers
// and has no option for this one, so it stays ours.
//
// It is applied when the response's header is written, so it overrides what an
// inner handler set (collage's "private, no-store" for a reader with a
// session), and deferred, so that a handler that wrote nothing or panicked
// still gets it.
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}
		nw := &noStoreWriter{ResponseWriter: w}
		defer nw.apply()
		next.ServeHTTP(nw, r)
	})
}

type noStoreWriter struct {
	http.ResponseWriter
	done bool
}

func (w *noStoreWriter) apply() {
	if !w.done {
		w.done = true
		w.Header().Set("Cache-Control", "no-store")
	}
}

func (w *noStoreWriter) WriteHeader(status int) {
	w.apply()
	w.ResponseWriter.WriteHeader(status)
}

func (w *noStoreWriter) Write(b []byte) (int, error) {
	w.apply()
	return w.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer when it can flush.
func (w *noStoreWriter) Flush() {
	w.apply()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *noStoreWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
