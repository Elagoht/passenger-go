package actions

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
	"passenger-go/internal/store"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// Confirm answers the recovery-key confirmation: when the key typed back
// matches (ignoring case, spaces and dashes) it writes the vault, opens a
// session and goes home.
func Confirm(d *web.Deps) *collage.Action {
	return collage.NewAction("setup-confirm").
		WithMethods(http.MethodPost).
		WithHandler(confirmHandler(d)).
		Build()
}

func confirmHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		r := rc.Request
		row, recovery, dek, ok := d.Pending.Get(web.PendingToken(r))
		defer dek.Wipe()
		if !ok { // expired between the guard and here
			return collage.SeeOtherPage(rc, "setup", nil)
		}
		if msg, ok, err := d.Throttle(r); err != nil {
			return nil, err
		} else if !ok {
			return throttled(rc, msg), nil
		}
		typed := vault.NormalizeRecoveryKey(r.PostFormValue("recovery"))
		if subtle.ConstantTimeCompare([]byte(typed), []byte(vault.NormalizeRecoveryKey(recovery))) != 1 {
			return refuse(rc, http.StatusUnprocessableEntity,
				"The recovery key does not match. Check what you wrote down and type it again."), nil
		}
		d.AttemptSucceeded(r)

		clearPending := web.ClearPendingCookie(r)
		if err := d.Store.CreateVault(ctx, row); errors.Is(err, store.ErrAlreadyInitialized) {
			d.Pending.Clear()
			res, err := collage.SeeOtherPage(rc, "login", nil)
			if err != nil {
				return nil, err
			}
			res.Header = web.CookieHeader(clearPending)
			return res, nil
		} else if err != nil {
			return nil, err
		}
		d.Pending.Clear()
		s, err := d.Sessions.Create(dek)
		if err != nil {
			return nil, err
		}
		res := collage.SeeOther("/")
		res.Header = web.CookieHeader(session.Cookie(r, s.Token), clearPending)
		return res, nil
	}
}
