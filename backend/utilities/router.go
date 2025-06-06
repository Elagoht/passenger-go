/**
 * Take a (writer, request) (error) function and return a http.HandlerFunc
 * that handles the request and returns the error.
 * Also determine the http method and the path of the request.
 * This must be attached to a chi.Mux.
 */

package utilities

import (
	"net/http"

	"github.com/go-chi/chi"
)

type Router struct {
	mux *chi.Mux
}

func NewRouter(mux *chi.Mux) *Router {
	return &Router{mux: mux}
}

// Mux returns the underlying chi.Mux instance
func (r *Router) Mux() *chi.Mux {
	return r.mux
}

type HandlerWithError func(http.ResponseWriter, *http.Request) error

func (r *Router) Handle(method string, pattern string, handler HandlerWithError) {
	wrappedHandler := func(w http.ResponseWriter, req *http.Request) {
		Logger.Printf("%s %s", req.Method, req.URL.Path)

		if err := handler(w, req); err != nil {
			HandleAPIError(w, err)
			return
		}
	}

	r.mux.Method(method, pattern, http.HandlerFunc(wrappedHandler))
}

// Convenience methods for common HTTP methods
func (r *Router) Get(pattern string, handler HandlerWithError) {
	r.Handle(http.MethodGet, pattern, handler)
}

func (r *Router) Post(pattern string, handler HandlerWithError) {
	r.Handle(http.MethodPost, pattern, handler)
}

func (r *Router) Put(pattern string, handler HandlerWithError) {
	r.Handle(http.MethodPut, pattern, handler)
}

func (r *Router) Delete(pattern string, handler HandlerWithError) {
	r.Handle(http.MethodDelete, pattern, handler)
}

func (r *Router) Patch(pattern string, handler HandlerWithError) {
	r.Handle(http.MethodPatch, pattern, handler)
}
