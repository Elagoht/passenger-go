package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/transfer"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/transfer"
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
