// Package pages assembles the setup, login and recovery pages.
package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/auth"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/auth"
	"passenger-go/internal/web"
)

// Setup is /setup, the first-run form; it exists only until the vault does.
func Setup(d *web.Deps) *collage.Page {
	return collage.NewPage("setup").
		WithLayouts(layouts.Auth()).
		WithContent(fragments.Setup(d)).
		WithPath("en", "/setup").
		WithActionFor(actions.Setup(d)).
		Dynamic().
		Build()
}
