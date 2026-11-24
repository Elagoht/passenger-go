package actions

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages"
	"passenger-go/internal/store"
	"passenger-go/internal/transfer"
	"passenger-go/internal/vault"
	"passenger-go/internal/web"
)

// maxImportBytes bounds an uploaded CSV, and with it the memory an import takes.
const maxImportBytes = 10 << 20

// Import stores each row of the uploaded CSV as an account and answers with
// the import page listing what could not be stored. A failed row never
// carries its passphrase.
func Import(d *web.Deps) *collage.Action {
	return collage.NewAction("import").
		WithMethods(http.MethodPost).
		WithMaxBodyBytes(maxImportBytes).
		WithHandler(func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			file, _, err := rc.Request.FormFile("file")
			if err != nil {
				return refuseImport(rc, "Choose a CSV file."), nil
			}
			defer file.Close()
			format, records, err := transfer.Import(file)
			if errors.Is(err, transfer.ErrUnknownFormat) {
				return refuseImport(rc, "This file is not a supported CSV (Firefox, Chromium or Passenger)."), nil
			}
			if err != nil {
				return refuseImport(rc, "This file could not be read as CSV."), nil
			}

			s, err := web.RequireSessionFrom(rc)
			if err != nil {
				return nil, err
			}
			defer s.DEK.Wipe()

			res := fragments.ImportResult{Format: format}
			for _, rec := range records {
				rec.Platform = strings.TrimSpace(rec.Platform)
				rec.Identifier = strings.TrimSpace(rec.Identifier)
				rec.URL = strings.TrimSpace(rec.URL)
				if reason := importProblem(rec); reason != "" {
					res.Failed = append(res.Failed, fragments.FailedRow{Platform: rec.Platform, Identifier: rec.Identifier, Reason: reason})
					continue
				}
				_, err := d.Store.CreateAccount(ctx, s.DEK, rec)
				if errors.Is(err, store.ErrDuplicate) {
					res.Failed = append(res.Failed, fragments.FailedRow{Platform: rec.Platform, Identifier: rec.Identifier, Reason: "duplicate"})
					continue
				}
				if err != nil {
					return nil, err
				}
				res.Imported++
			}
			fragments.Imported.Set(rc, res)
			return collage.RenderPage(rc.Page), nil
		}).
		Build()
}

func importProblem(r vault.Record) string {
	switch {
	case r.Platform == "":
		return "missing platform"
	case r.Identifier == "":
		return "missing identifier"
	case r.Passphrase == "":
		return "missing passphrase"
	}
	return ""
}

func refuseImport(rc *collage.RenderContext, msg string) *collage.ActionResult {
	fragments.FormError.Set(rc, msg)
	return &collage.ActionResult{Status: http.StatusUnprocessableEntity, Page: rc.Page}
}
