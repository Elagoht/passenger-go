package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/actions"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages"
	"passenger-go/internal/web"
)

// Export is /export, the CSV download behind the master passphrase.
func Export(d *web.Deps) *collage.Page {
	return collage.NewPage("export").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Export()).
		WithPath("en", "/export").
		WithActionFor(actions.Export(d)).
		Dynamic().
		Build()
}
