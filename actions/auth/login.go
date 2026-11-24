package actions

import (
	"context"
	"errors"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// Login answers the unlock form: the passphrase is right when it unwraps the
// DEK, which then lives in the new session.
func Login(d *web.Deps) *collage.Action {
	return collage.NewAction("login").
		WithMethods(http.MethodPost).
		WithHandler(loginHandler(d)).
		Build()
}

func loginHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		r := rc.Request
		if msg, ok, err := d.Throttle(r); err != nil {
			return nil, err
		} else if !ok {
			return throttled(rc, msg), nil
		}
		row, err := d.Store.Vault(ctx)
		if err != nil {
			return nil, err
		}
		dek, err := web.Unwrap(ctx, r.PostFormValue("passphrase"), row.Pass, row.KDF, vault.AADPass)
		if errors.Is(err, vault.ErrWrongSecret) {
			return refuse(rc, http.StatusUnprocessableEntity, "Wrong passphrase."), nil
		}
		if err != nil {
			return nil, err
		}
		defer dek.Wipe()
		d.AttemptSucceeded(r)

		s, err := d.Sessions.Create(dek)
		if err != nil {
			return nil, err
		}
		s.DEK.Wipe() // our copy; the stored session keeps its own
		res := collage.SeeOther(web.SafeNext(r.PostFormValue("next")))
		res.Header = web.CookieHeader(session.Cookie(r, s.Token))
		return res, nil
	}
}
