package main

import (
	"github.com/Elagoht/collage/pkg/collage"

	authactions "passenger-go/actions/auth"
	"passenger-go/internal/web"
	"passenger-go/pages"
	authpages "passenger-go/pages/auth"
)

// register adds every page, document and action to app.
func register(app *collage.App, deps *web.Deps) error {
	return app.Register(
		pages.Home(deps),
		authpages.Setup(deps),
		authpages.SetupConfirm(deps),
		authpages.Login(deps),
		authpages.Recover(deps),
		authactions.Logout(deps),
	)
}
