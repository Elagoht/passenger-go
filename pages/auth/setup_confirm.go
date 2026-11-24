package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/auth"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/auth"
	"passenger-go/internal/web"
)

// SetupConfirm is /setup/confirm, where the recovery key is shown and typed
// back before the vault is written.
func SetupConfirm(d *web.Deps) *collage.Page {
	return collage.NewPage("setup-confirm").
		WithLayouts(layouts.Auth()).
		WithContent(fragments.SetupConfirm(d)).
		WithPath("en", "/setup/confirm").
		WithActionFor(actions.Confirm(d)).
		Dynamic().
		Build()
}
