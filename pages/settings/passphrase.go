// Package pages assembles the settings pages.
package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/settings"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/settings"
	"passenger-go/internal/web"
)

// Passphrase is /settings/passphrase, the master passphrase change.
func Passphrase(d *web.Deps) *collage.Page {
	return collage.NewPage("settings-passphrase").
		WithLayouts(layouts.App(d)).
		WithContent(fragments.Passphrase()).
		WithPath("en", "/settings/passphrase").
		WithActionFor(actions.Passphrase(d)).
		Dynamic().
		Build()
}
