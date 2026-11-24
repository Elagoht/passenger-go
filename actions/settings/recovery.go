package actions

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages/settings"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// Recovery issues a new recovery key for the DEK and answers with the page
// showing it once. The old key stops working.
func Recovery(d *web.Deps) *collage.Action {
	return collage.NewAction("settings-recovery").
		WithMethods(http.MethodPost).
		WithHandler(func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			d.WrapMu.Lock()
			defer d.WrapMu.Unlock()
			dek, kdf, refused, err := unlock(ctx, d, rc)
			if err != nil || refused != nil {
				return refused, err
			}
			defer dek.Wipe()

			key, err := vault.NewRecoveryKey()
			if err != nil {
				return nil, err
			}
			wrapped, err := web.Wrap(ctx, vault.NormalizeRecoveryKey(key), dek, kdf, vault.AADRecovery)
			if err != nil {
				return nil, err
			}
			if err := d.Store.UpdateRecoveryWrap(ctx, wrapped); err != nil {
				return nil, err
			}
			fragments.NewRecoveryKey.Set(rc, key)
			return collage.RenderPage(rc.Page), nil
		}).
		Build()
}
