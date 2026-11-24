package web

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
)

// Guard targets are literal paths: the pages are registered by name in later
// tasks, and a literal cannot fail at request time.
const (
	pathSetup = "/setup"
	pathLogin = "/login"
	pathHome  = "/"
)

func seeOther(location string) *collage.GuardDecision {
	return &collage.GuardDecision{Status: http.StatusSeeOther, Location: location}
}

// RequireSession lets through requests carrying a live session. Without a
// vault it sends the reader to /setup; without a session to /login, with the
// requested path in next for a GET or HEAD of anything but the home page.
// Another method's path is not one to return to: after login the browser
// would GET it.
func (d *Deps) RequireSession(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
	ok, err := d.Store.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return seeOther(pathSetup), nil
	}
	if _, ok := session.FromContext(ctx); ok {
		return nil, nil
	}
	loc := pathLogin
	if next := r.URL.RequestURI(); (r.Method == http.MethodGet || r.Method == http.MethodHead) && next != "/" {
		loc += "?" + url.Values{"next": {next}}.Encode()
	}
	return seeOther(loc), nil
}

// RequireGuest lets through readers with no session on an initialized vault.
func (d *Deps) RequireGuest(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
	ok, err := d.Store.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return seeOther(pathSetup), nil
	}
	if _, ok := session.FromContext(ctx); ok {
		return seeOther(pathHome), nil
	}
	return nil, nil
}

// RequireUninitialized lets through requests only while no vault exists.
func (d *Deps) RequireUninitialized(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
	ok, err := d.Store.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	if ok {
		return seeOther(pathLogin), nil
	}
	return nil, nil
}
