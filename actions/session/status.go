// Package session holds the two JSON endpoints behind the expiry banner.
package session

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

type expiryResponse struct {
	ExpiresAt int64 `json:"expiresAt"`
}

// answer reports the session's expiry as unix milliseconds, or 401 with
// expiresAt 0 when there is none. These URLs answer scripts, so they say so
// themselves instead of redirecting through RequireSession.
func answer(d *web.Deps, rc *collage.RenderContext) (*collage.ActionResult, error) {
	s := web.SessionFrom(rc)
	if s == nil {
		return collage.JSON(http.StatusUnauthorized, expiryResponse{})
	}
	return collage.JSON(http.StatusOK, expiryResponse{ExpiresAt: d.Sessions.ExpiresAt(s).UnixMilli()})
}

// Status is GET /session. The middleware only peeks at this path, so polling
// it never extends the session.
func Status(d *web.Deps) *collage.Action {
	return collage.NewAction("session").
		WithPath("en", "/session").
		WithMethods(http.MethodGet, http.MethodHead).
		WithHandler(func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			return answer(d, rc)
		}).
		Build()
}
