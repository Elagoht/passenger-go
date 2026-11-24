package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/auth"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/auth"
	"passenger-go/internal/web"
)

// Recover is /recover: a new passphrase in exchange for the recovery key.
func Recover(d *web.Deps) *collage.Page {
	return collage.NewPage("recover").
		WithLayouts(layouts.Auth()).
		WithContent(fragments.Recover(d)).
		WithPath("en", "/recover").
		WithActionFor(actions.Recover(d)).
		Dynamic().
		Build()
}
