package main

import (
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/fragments/layouts"
	"passenger-go/internal/web"
)

// register adds every page, document and action to app.
func register(app *collage.App, deps *web.Deps) error {
	return app.Register(home(deps))
}

// home is a temporary page proving the App layout builds and guards; replaced
// by the account list later.
func home(deps *web.Deps) *collage.Page {
	content := collage.NewInlineFragment("home", `<h1>Passenger</h1>`).Build()
	return collage.NewPage("home").
		WithLayouts(layouts.App(deps)).
		WithContent(content).
		WithPath("en", "/").
		Build()
}
