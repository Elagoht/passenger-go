package main

import (
	session "github.com/Elagoht/collage-session"
	"github.com/Elagoht/collage/pkg/collage"

	accountactions "passenger-go/actions/accounts"
	authactions "passenger-go/actions/auth"
	sessionactions "passenger-go/actions/session"
	"passenger-go/internal/web"
	"passenger-go/middleware"
	accountpages "passenger-go/pages/accounts"
	authpages "passenger-go/pages/auth"
	generatorpages "passenger-go/pages/generator"
	settingspages "passenger-go/pages/settings"
	transferpages "passenger-go/pages/transfer"
)

// register adds the application's middleware, the session plugin, and every
// page, document and action to app.
func register(app *collage.App, deps *web.Deps, sessions *session.Plugin) error {
	if err := app.Use(middleware.NoStore); err != nil {
		return err
	}
	// The session plugin sits between SessionCookie, which must be outside it
	// to adjust the cookie it writes, and VaultSession, which reads its
	// session: a plugin registered here is inside the middleware added before
	// it.
	if err := app.Use(middleware.SessionCookie); err != nil {
		return err
	}
	if err := app.RegisterPlugin(sessions); err != nil {
		return err
	}
	if err := app.Use(middleware.VaultSession(deps.Sessions)); err != nil {
		return err
	}

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
		sessionactions.Status(deps),
		sessionactions.Extend(deps),
		generatorpages.Generate(deps),
		transferpages.Import(deps),
		transferpages.Export(deps),
		settingspages.Passphrase(deps),
		settingspages.Recovery(deps),
	)
}
