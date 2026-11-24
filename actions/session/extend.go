package session

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
	"passenger-go/utils"
)

// Extend is POST /session/extend. middleware.VaultSession has already slid
// the idle window, of the DEK and of the cookie alike, by the time the handler
// runs, so answering with the expiry is all that is left. The forgery check
// wants the token in X-CSRF-Token.
func Extend(d *web.Deps) *collage.Action {
	return collage.NewAction("session-extend").
		WithPath("en", "/session/extend").
		WithMethods(http.MethodPost).
		WithHandler(extendHandler(d)).
		Build()
}

// extendHandler reports the session's new end.
func extendHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		return utils.SessionExpiry(d, rc)
	}
}
