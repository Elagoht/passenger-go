package utils

import (
	"context"
	"errors"

	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// Unlock checks the master passphrase a settings form sent as "current" the
// way login does (throttled, in a KDF slot) and returns the DEK it unwraps,
// which the caller must wipe, and the vault's KDF parameters. When the
// passphrase is refused the DEK is zero and the result is the answer to send.
// The caller holds d.WrapMu, so the row read here is still the stored one when
// the caller writes its new wrap.
func Unlock(ctx context.Context, d *web.Deps, rc *collage.RenderContext, v *validate.Validator) (vault.DEK, vault.KDFParams, *collage.ActionResult, error) {
	r := rc.Request
	if msg, ok, err := d.Throttle(r); err != nil {
		return vault.DEK{}, vault.KDFParams{}, nil, err
	} else if !ok {
		return vault.DEK{}, vault.KDFParams{}, Throttled(rc, v, "current", msg), nil
	}
	row, err := d.Store.Vault(ctx)
	if err != nil {
		return vault.DEK{}, vault.KDFParams{}, nil, err
	}
	dek, err := web.Unwrap(ctx, v.Value("current"), row.Pass, row.KDF, vault.AADPass)
	if errors.Is(err, vault.ErrWrongSecret) {
		return vault.DEK{}, vault.KDFParams{}, Refuse(rc, v, "current", "Wrong passphrase."), nil
	}
	if err != nil {
		return vault.DEK{}, vault.KDFParams{}, nil, err
	}
	d.AttemptSucceeded(r)
	return dek, row.KDF, nil, nil
}
