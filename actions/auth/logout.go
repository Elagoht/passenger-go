package actions

import (
	"context"
	"net/http"

	session "github.com/Elagoht/collage-session"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/guards"
	"passenger-go/internal/web"
)

// Logout ends the session (wiping its DEK) and clears the cookie. It has a
// URL of its own: the App layout's logout form posts to it from every page.
func Logout(d *web.Deps) *collage.Action {
	return collage.NewAction("logout").
		WithPath("en", "/logout").
		WithMethods(http.MethodPost).
		WithGuard(guards.SignedIn(d)).
		WithHandler(logoutHandler(d)).
		Build()
}

// logoutHandler drops the handle's DEK and the cookie, and goes to login.
func logoutHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		if s := web.SessionFrom(rc); s != nil {
			d.Sessions.Delete(s.Handle)
		}
		session.Get(rc).Clear()
		return collage.SeeOtherPage(rc, "login", nil)
	}
}
