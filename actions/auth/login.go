// Package actions holds the form handlers of the setup, login, logout and
// recovery pages.
package actions

import (
	"context"
	"errors"
	"net/http"

	session "github.com/Elagoht/collage-session"
	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/vault"
	"passenger-go/internal/web"
	"passenger-go/utils"
)

// Login answers the unlock form: the passphrase is right when it unwraps the
// DEK, which then lives in the new session.
func Login(d *web.Deps) *collage.Action {
	return collage.NewAction("login").
		WithMethods(http.MethodPost).
		WithHandler(loginHandler(d)).
		Build()
}

// loginHandler checks the passphrase behind the backoff and, when it is
// right, signs the reader in and sends them to next.
func loginHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		r := rc.Request
		v := validate.Form(rc)
		if msg, ok, err := d.Throttle(r); err != nil {
			return nil, err
		} else if !ok {
			return utils.Throttled(rc, v, "passphrase", msg), nil
		}
		row, err := d.Store.Vault(ctx)
		if err != nil {
			return nil, err
		}
		dek, err := web.Unwrap(ctx, v.Value("passphrase"), row.Pass, row.KDF, vault.AADPass)
		if errors.Is(err, vault.ErrWrongSecret) {
			return utils.Refuse(rc, v, "passphrase", "Wrong passphrase."), nil
		}
		if err != nil {
			return nil, err
		}
		defer dek.Wipe()
		d.AttemptSucceeded(r)

		if err := signIn(rc, d, dek); err != nil {
			return nil, err
		}
		return collage.SeeOther(utils.SafeRedirect(v.Value("next"), "/")), nil
	}
}

// signIn opens a session on dek: the manager keeps the DEK under a new handle,
// and the elagoht/session cookie, given a new id and lifetime, holds the
// handle.
func signIn(rc *collage.RenderContext, d *web.Deps, dek vault.DEK) error {
	s, err := d.Sessions.Create(dek)
	if err != nil {
		return err
	}
	s.DEK.Wipe() // our copy; the stored session keeps its own
	cs := session.Get(rc)
	cs.Regenerate()
	if err := cs.Set(session.UserKey, s.Handle); err != nil {
		d.Sessions.Delete(s.Handle)
		return err
	}
	return nil
}
