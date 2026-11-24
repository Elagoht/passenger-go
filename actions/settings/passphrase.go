// Package actions holds the handlers of the settings pages.
package actions

import (
	"context"
	"errors"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages/settings"
	"passenger-go/internal/session"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

func refuse(rc *collage.RenderContext, status int, msg string) *collage.ActionResult {
	fragments.FormError.Set(rc, msg)
	return &collage.ActionResult{Status: status, Page: rc.Page}
}

// unlock checks the current passphrase the way login does (throttled, in a KDF
// slot) and returns the DEK it unwraps, which the caller must wipe, and the
// vault's KDF parameters. When the passphrase is refused the DEK is zero and
// the result is the answer to send. The caller holds d.WrapMu, so the row read
// here is still the stored one when the caller writes its new wrap.
func unlock(ctx context.Context, d *web.Deps, rc *collage.RenderContext) (vault.DEK, vault.KDFParams, *collage.ActionResult, error) {
	r := rc.Request
	if msg, ok, err := d.Throttle(r); err != nil {
		return vault.DEK{}, vault.KDFParams{}, nil, err
	} else if !ok {
		return vault.DEK{}, vault.KDFParams{}, refuse(rc, http.StatusTooManyRequests, msg), nil
	}
	row, err := d.Store.Vault(ctx)
	if err != nil {
		return vault.DEK{}, vault.KDFParams{}, nil, err
	}
	dek, err := web.Unwrap(ctx, r.PostFormValue("current"), row.Pass, row.KDF, vault.AADPass)
	if errors.Is(err, vault.ErrWrongSecret) {
		return vault.DEK{}, vault.KDFParams{}, refuse(rc, http.StatusUnprocessableEntity, "Wrong passphrase."), nil
	}
	if err != nil {
		return vault.DEK{}, vault.KDFParams{}, nil, err
	}
	d.AttemptSucceeded(r)
	return dek, row.KDF, nil, nil
}

// Passphrase changes the master passphrase: the DEK is wrapped again under the
// new one, every other session ends and this one gets a new token.
func Passphrase(d *web.Deps) *collage.Action {
	return collage.NewAction("settings-passphrase").
		WithMethods(http.MethodPost).
		WithHandler(func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			r := rc.Request
			pass := r.PostFormValue("passphrase")
			if msg := web.CheckNewPassphrase(pass, r.PostFormValue("confirm")); msg != "" {
				return refuse(rc, http.StatusUnprocessableEntity, msg), nil
			}
			s, err := web.RequireSessionFrom(rc)
			if err != nil {
				return nil, err
			}
			defer s.DEK.Wipe()

			d.WrapMu.Lock()
			defer d.WrapMu.Unlock()
			dek, kdf, refused, err := unlock(ctx, d, rc)
			if err != nil || refused != nil {
				return refused, err
			}
			defer dek.Wipe()

			wrapped, err := web.Wrap(ctx, pass, dek, kdf, vault.AADPass)
			if err != nil {
				return nil, err
			}
			if err := d.Store.UpdatePassWrap(ctx, wrapped); err != nil {
				return nil, err
			}
			rotated, err := d.Sessions.Rotate(s.Token)
			if err != nil {
				// The passphrase changed: no session opened under the old one
				// may survive.
				d.Sessions.DeleteAll()
				return nil, err
			}
			defer rotated.DEK.Wipe()

			res := collage.SeeOther("/settings/passphrase?saved=1")
			res.Header = web.CookieHeader(session.Cookie(r, rotated.Token))
			return res, nil
		}).
		Build()
}
