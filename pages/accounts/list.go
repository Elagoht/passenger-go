// Package pages assembles the account pages.
package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/accounts"
	"passenger-go/internal/web"
)

// List is "/", the account list with its search.
func List(d *web.Deps) *collage.Page {
	return collage.NewPage("accounts").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.List(d)).
		WithPath("en", "/").
		Dynamic().
		Build()
}
