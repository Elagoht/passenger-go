// Package actions holds the handlers of the import and export pages.
package actions

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/transfer"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
	"passenger-go/utils"
)

// Export answers the export form with the vault as a CSV download, once the
// master passphrase has been typed again.
func Export(d *web.Deps) *collage.Action {
	return collage.NewAction("export").
		WithMethods(http.MethodPost).
		WithHandler(exportHandler(d)).
		Build()
}

// exportHandler checks the passphrase behind the backoff and answers with
// the CSV.
func exportHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		r := rc.Request
		v := utils.PostForm(rc)
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
	}
}
