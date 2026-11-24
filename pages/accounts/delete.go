package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/accounts"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/accounts"
	"passenger-go/internal/web"
)

// Delete is /accounts/{id}/delete: a confirmation page whose POST deletes.
func Delete(d *web.Deps) *collage.Page {
	return collage.NewPage("account-delete").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Delete(d)).
		WithPath("en", "/accounts/{id}/delete").
		WithActionFor(actions.Delete(d)).
		Dynamic().
		Build()
}
