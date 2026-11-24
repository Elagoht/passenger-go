package actions

import (
	"context"
	"errors"
	"net/http"

	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages/auth"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
	"passenger-go/utils"
	"passenger-go/validations"
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

// recoverHandler checks the new passphrase first, then the key behind the
// backoff, all under the wrap mutex from reading the vault row on.
func recoverHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		r := rc.Request
		v := utils.PostForm(rc)
		validations.Recover(v)
		if !v.Valid() {
			return validate.Refuse(rc, v, rc.Page), nil
		}
		if msg, ok, err := d.Throttle(r); err != nil {
			return nil, err
		} else if !ok {
			return utils.Throttled(rc, v, "recovery", msg), nil
		}
		d.WrapMu.Lock()
		defer d.WrapMu.Unlock()
		row, err := d.Store.Vault(ctx)
		if err != nil {
			return nil, err
		}
		key := vault.NormalizeRecoveryKey(v.Value("recovery"))
		dek, err := web.Unwrap(ctx, key, row.Recovery, row.KDF, vault.AADRecovery)
		if errors.Is(err, vault.ErrWrongSecret) {
			return utils.Refuse(rc, v, "recovery", "Wrong recovery key."), nil
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
		passWrap, err := web.Wrap(ctx, v.Value("passphrase"), dek, row.KDF, vault.AADPass)
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
