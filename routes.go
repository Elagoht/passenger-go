package main

import (
	"github.com/Elagoht/collage/pkg/collage"

	accountactions "passenger-go/actions/accounts"
	authactions "passenger-go/actions/auth"
	"passenger-go/internal/web"
	"passenger-go/pages"
	accountpages "passenger-go/pages/accounts"
	authpages "passenger-go/pages/auth"
	settingspages "passenger-go/pages/settings"
)

// register adds every page, document and action to app.
func register(app *collage.App, deps *web.Deps) error {
	return app.Register(
		authpages.Setup(deps),
		authpages.SetupConfirm(deps),
		authpages.Login(deps),
		authpages.Recover(deps),
		authactions.Logout(deps),
		accountpages.List(deps),
		accountpages.New(deps),
		accountpages.Detail(deps),
		accountpages.Delete(deps),
		accountactions.Passphrase(deps),
		pages.Generate(deps),
		pages.Import(deps),
		pages.Export(deps),
		settingspages.Passphrase(deps),
		settingspages.Recovery(deps),
	)
}
