// Package session holds the two JSON endpoints behind the expiry banner.
package session

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
	"passenger-go/utils"
)

// Status is GET /session. middleware.VaultSession only peeks at this path, and
// middleware.QuietSession keeps the cookie from being refreshed on it, so
// polling it never extends the session.
func Status(d *web.Deps) *collage.Action {
	return collage.NewAction("session").
		WithPath("en", "/session").
		WithMethods(http.MethodGet, http.MethodHead).
		WithHandler(statusHandler(d)).
		Build()
}

// statusHandler reports the session's end.
func statusHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		return utils.SessionExpiry(d, rc)
	}
}
