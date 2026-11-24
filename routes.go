package main

import "github.com/Elagoht/collage/pkg/collage"

// register adds every page, document and action to app.
func register(app *collage.App) error {
	return app.Register(home())
}

// home is a temporary page proving the skeleton boots; replaced later.
func home() *collage.Page {
	layout := collage.NewFragment("layout", "layouts/base.html").
		WithTitle("Passenger").
		Build()
	content := collage.NewInlineFragment("home", `<main><h1>Passenger</h1></main>`).Build()
	return collage.NewPage("home").
		WithLayouts(layout).
		WithContent(content).
		WithPath("en", "/").
		Build()
}
