// Package pages assembles the import and export pages.
package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/transfer"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/transfer"
	"passenger-go/internal/web"
)

// Import is /import, the CSV upload.
func Import(d *web.Deps) *collage.Page {
	return collage.NewPage("import").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Import()).
		WithPath("en", "/import").
		WithActionFor(actions.Import(d)).
		Dynamic().
		Build()
}
