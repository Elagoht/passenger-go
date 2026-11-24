package session

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// Extend is POST /session/extend. The middleware has already slid the idle
// window by the time the handler runs, so answering with the expiry is all
// that is left. The forgery check wants the token in X-CSRF-Token.
func Extend(d *web.Deps) *collage.Action {
	return collage.NewAction("session-extend").
		WithPath("en", "/session/extend").
		WithMethods(http.MethodPost).
		WithHandler(func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			return answer(d, rc)
		}).
		Build()
}
