// Package pages assembles the generator page.
package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/generator"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/generator"
	"passenger-go/internal/web"
)

// Generate is /generate, the passphrase generator.
func Generate(d *web.Deps) *collage.Page {
	result := fragments.Result()
	return collage.NewPage("generate").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Generate(result)).
		WithPath("en", "/generate").
		WithActionFor(actions.Generate(d, result)).
		Dynamic().
		Build()
}
