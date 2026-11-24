package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/actions"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages"
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
