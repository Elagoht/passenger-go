// Package pages assembles the pages behind the vault.
package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/fragments/layouts"
	"passenger-go/internal/web"
)

// Home is "/". For now a placeholder behind the App layout (and so behind a
// session); the account list replaces it.
func Home(d *web.Deps) *collage.Page {
	content := collage.NewInlineFragment("home", `<h1>Accounts</h1>
<p>No accounts yet.</p>`).
		WithTitle("Accounts").
		Build()
	return collage.NewPage("home").
		WithLayouts(layouts.App(d)).
		WithContent(content).
		WithPath("en", "/").
		Build()
}
