// Package actions holds the handlers of the settings pages.
package actions

import (
	"context"
	"net/http"

	flash "github.com/Elagoht/collage-flash"
	session "github.com/Elagoht/collage-session"
	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/vault"
	"passenger-go/internal/web"
	"passenger-go/utils"
	"passenger-go/validations"
)

// Passphrase changes the master passphrase: the DEK is wrapped again under the
// new one, every other session ends and this one gets a new handle.
func Passphrase(d *web.Deps) *collage.Action {
	return collage.NewAction("settings-passphrase").
		WithMethods(http.MethodPost).
		WithHandler(passphraseHandler(d)).
		Build()
}

// passphraseHandler checks the new passphrase, then the current one behind
// the backoff and under the wrap mutex, stores the new wrap, rotates the
// session and comes back to the page with a flash message.
func passphraseHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		v := validate.Form(rc)
		validations.Passphrase(v)
		if !v.Valid() {
			return validate.Refuse(rc, v, rc.Page), nil
		}
		s, err := web.RequireSessionFrom(rc)
		if err != nil {
			return nil, err
		}
		defer s.DEK.Wipe()

		d.WrapMu.Lock()
		defer d.WrapMu.Unlock()
		dek, kdf, refused, err := utils.Unlock(ctx, d, rc, v)
		if err != nil || refused != nil {
			return refused, err
		}
		defer dek.Wipe()

		wrapped, err := web.Wrap(ctx, v.Value("passphrase"), dek, kdf, vault.AADPass)
		if err != nil {
			return nil, err
		}
		if err := d.Store.UpdatePassWrap(ctx, wrapped); err != nil {
			return nil, err
		}
		rotated, err := d.Sessions.Rotate(s.Handle)
		if err != nil {
			// The passphrase changed: no session opened under the old one
			// may survive.
			d.Sessions.DeleteAll()
			return nil, err
		}
		defer rotated.DEK.Wipe()
		// The cookie keeps its id and creation time, as the manager keeps the
		// session's: both still end at the same moment. Only the handle, the
		// part that can be revoked, changes.
		if err := session.Get(rc).Set(session.UserKey, rotated.Handle); err != nil {
			d.Sessions.DeleteAll()
			return nil, err
		}

		flash.Add(rc, flash.Success, "Passphrase changed.")
		return collage.SeeOtherPage(rc, "settings-passphrase", nil)
	}
}
