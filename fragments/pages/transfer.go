package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"
)

// FailedRow is an imported row that was not stored. It carries no passphrase.
type FailedRow struct {
	Platform   string
	Identifier string
	Reason     string
}

// ImportResult is what an import did, shown on the page that answers it.
type ImportResult struct {
	Format   string
	Imported int
	Failed   []FailedRow
}

// Imported carries the result of an import to the page.
var Imported = collage.NewKey[ImportResult]("import-result")

type importView struct {
	Error  string
	Result *ImportResult
}

// Import is the CSV import form and, once submitted, its outcome.
func Import() *collage.Fragment {
	return collage.NewFragment("import", "pages/import.html").
		WithTitle("Import - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (importView, error) {
			msg, _ := FormError.Get(rc)
			v := importView{Error: msg}
			if res, ok := Imported.Get(rc); ok {
				v.Result = &res
			}
			return v, nil
		})).
		Build()
}

type exportView struct{ Error string }

// Export asks for the master passphrase before the CSV is downloaded.
func Export() *collage.Fragment {
	return collage.NewFragment("export", "pages/export.html").
		WithTitle("Export - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (exportView, error) {
			msg, _ := FormError.Get(rc)
			return exportView{Error: msg}, nil
		})).
		Build()
}
