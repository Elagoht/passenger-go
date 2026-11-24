package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/accounts"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/accounts"
	"passenger-go/internal/web"
)

// New is /accounts/new, the form for a new account.
func New(d *web.Deps) *collage.Page {
	return collage.NewPage("account-new").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.New()).
		WithPath("en", "/accounts/new").
		WithActionFor(actions.Create(d)).
		Dynamic().
		Build()
}
