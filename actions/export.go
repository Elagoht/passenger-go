package actions

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages"
	"passenger-go/internal/transfer"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// Export answers the export form with the vault as a CSV download, once the
// master passphrase has been typed again.
func Export(d *web.Deps) *collage.Action {
	return collage.NewAction("export").
		WithMethods(http.MethodPost).
		WithHandler(func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			r := rc.Request
			if msg, ok, err := d.Throttle(r); err != nil {
				return nil, err
			} else if !ok {
				return exportRefused(rc, http.StatusTooManyRequests, msg), nil
			}
			row, err := d.Store.Vault(ctx)
			if err != nil {
				return nil, err
			}
			dek, err := web.Unwrap(ctx, r.PostFormValue("passphrase"), row.Pass, row.KDF, vault.AADPass)
			if errors.Is(err, vault.ErrWrongSecret) {
				return exportRefused(rc, http.StatusUnprocessableEntity, "Wrong passphrase."), nil
			}
			if err != nil {
				return nil, err
			}
			defer dek.Wipe()
			d.AttemptSucceeded(r)

			accounts, err := d.Store.ListAccounts(ctx, dek)
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			if err := transfer.Export(&buf, accounts); err != nil {
				return nil, err
			}
			h := http.Header{}
			h.Set("Content-Disposition", `attachment; filename="passenger-export-`+d.Now().Format("2006-01-02")+`.csv"`)
			return &collage.ActionResult{Body: buf.Bytes(), ContentType: "text/csv; charset=utf-8", Header: h}, nil
		}).
		Build()
}

func exportRefused(rc *collage.RenderContext, status int, msg string) *collage.ActionResult {
	fragments.FormError.Set(rc, msg)
	return &collage.ActionResult{Status: status, Page: rc.Page}
}
