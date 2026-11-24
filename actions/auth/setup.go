package actions

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// Setup answers the setup form: with the right setup code and an acceptable
// passphrase it prepares the vault in memory (nothing is written yet) and
// sends the reader to confirm the recovery key.
func Setup(d *web.Deps) *collage.Action {
	return collage.NewAction("setup").
		WithMethods(http.MethodPost).
		WithHandler(setupHandler(d)).
		Build()
}

func setupHandler(d *web.Deps) collage.ActionHandlerFunc {
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
		if !d.SetupCode.Check(r.PostFormValue("code")) {
			return refuse(rc, http.StatusUnprocessableEntity, "Wrong setup code."), nil
		}
		d.AttemptSucceeded(r)

		dek, err := vault.NewDEK()
		if err != nil {
			return nil, err
		}
		defer dek.Wipe()
		recovery, err := vault.NewRecoveryKey()
		if err != nil {
			return nil, err
		}
		passWrap, err := web.Wrap(ctx, pass, dek, d.KDF, vault.AADPass)
		if err != nil {
			return nil, err
		}
		recWrap, err := web.Wrap(ctx, vault.NormalizeRecoveryKey(recovery), dek, d.KDF, vault.AADRecovery)
		if err != nil {
			return nil, err
		}
		token, err := d.Pending.Put(dek, store.VaultRow{KDF: d.KDF, Pass: passWrap, Recovery: recWrap}, recovery)
		if err != nil {
			return nil, err
		}
		res, err := collage.SeeOtherPage(rc, "setup-confirm", nil)
		if err != nil {
			return nil, err
		}
		res.Header = web.CookieHeader(web.PendingCookie(r, token))
		return res, nil
	}
}
