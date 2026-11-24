package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/accounts"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/accounts"
	"passenger-go/internal/web"
)

// Detail is /accounts/{id}: view and edit one account.
func Detail(d *web.Deps) *collage.Page {
	return collage.NewPage("account").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Detail(d)).
		WithPath("en", "/accounts/{id}").
		WithActionFor(actions.Update(d)).
		Dynamic().
		Build()
}
