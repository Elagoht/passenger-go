package actions

import (
	"context"
	"errors"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages/auth"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// Recover answers the recovery form: the recovery key unwraps the DEK, which
// is wrapped again under the new passphrase and a new recovery key (the used
// one counts as exposed). Every session ends, and the page shows the new key
// once.
func Recover(d *web.Deps) *collage.Action {
	return collage.NewAction("recover").
		WithMethods(http.MethodPost).
		WithHandler(recoverHandler(d)).
		Build()
}

func recoverHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		r := rc.Request
		pass := r.PostFormValue("passphrase")
		if msg := web.CheckNewPassphrase(pass, r.PostFormValue("confirm")); msg != "" {
			return refuse(rc, http.StatusUnprocessableEntity, msg), nil
		}
		if msg, ok, err := d.Throttle(r); err != nil {
			return nil, err
		} else if !ok {
			return throttled(rc, msg), nil
		}
		d.WrapMu.Lock()
		defer d.WrapMu.Unlock()
		row, err := d.Store.Vault(ctx)
		if err != nil {
			return nil, err
		}
		key := vault.NormalizeRecoveryKey(r.PostFormValue("recovery"))
		dek, err := web.Unwrap(ctx, key, row.Recovery, row.KDF, vault.AADRecovery)
		if errors.Is(err, vault.ErrWrongSecret) {
			return refuse(rc, http.StatusUnprocessableEntity, "Wrong recovery key."), nil
		}
		if err != nil {
			return nil, err
		}
		defer dek.Wipe()
		d.AttemptSucceeded(r)

		newKey, err := vault.NewRecoveryKey()
		if err != nil {
			return nil, err
		}
		passWrap, err := web.Wrap(ctx, pass, dek, row.KDF, vault.AADPass)
		if err != nil {
			return nil, err
		}
		recWrap, err := web.Wrap(ctx, vault.NormalizeRecoveryKey(newKey), dek, row.KDF, vault.AADRecovery)
		if err != nil {
			return nil, err
		}
		if err := d.Store.UpdateWraps(ctx, passWrap, recWrap); err != nil {
			return nil, err
		}
		d.Sessions.DeleteAll()

		fragments.NewRecoveryKey.Set(rc, newKey)
		return collage.RenderPage(rc.Page), nil
	}
}
