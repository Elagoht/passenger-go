package actions

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
	"passenger-go/internal/web"
)

// Logout ends the session (wiping its DEK) and clears the cookie. It has a
// URL of its own: the App layout's logout form posts to it from every page.
func Logout(d *web.Deps) *collage.Action {
	return collage.NewAction("logout").
		WithPath("en", "/logout").
		WithMethods(http.MethodPost).
		WithGuard(d.RequireSession).
		WithHandler(func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			if s := web.SessionFrom(rc); s != nil {
				d.Sessions.Delete(s.Token)
			}
			res, err := collage.SeeOtherPage(rc, "login", nil)
			if err != nil {
				return nil, err
			}
			res.Header = web.CookieHeader(session.ExpiredCookie(rc.Request))
			return res, nil
		}).
		Build()
}
