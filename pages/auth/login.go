package pages

import (
	"github.com/Elagoht/collage/pkg/collage"

	actions "passenger-go/actions/auth"
	"passenger-go/fragments/layouts"
	fragments "passenger-go/fragments/pages/auth"
	"passenger-go/internal/web"
)

// Login is /login, the unlock form.
func Login(d *web.Deps) *collage.Page {
	return collage.NewPage("login").
		WithLayouts(layouts.Auth()).
		WithContent(fragments.Login(d)).
		WithPath("en", "/login").
		WithActionFor(actions.Login(d)).
		Dynamic().
		Build()
}
