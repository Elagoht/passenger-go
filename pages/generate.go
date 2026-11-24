// Package pages assembles the generator page.
package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/actions"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages"
	"passenger-go/internal/web"
)

// Generate is /generate, the passphrase generator.
func Generate(d *web.Deps) *collage.Page {
	return collage.NewPage("generate").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Generate()).
		WithPath("en", "/generate").
		WithActionFor(actions.Generate(d)).
		Dynamic().
		Build()
}
