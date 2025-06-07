package backend

import (
	"passenger-go/backend/controllers"

	"github.com/go-chi/chi"
)

var authController = controllers.NewAuthController()
var accountsController = controllers.NewAccountsController()

func MountBackend(router *chi.Mux) *chi.Mux {
	apiRouter := chi.NewRouter()

	authController.MountAuthRouter(apiRouter)
	accountsController.MountAccountsRouter(apiRouter)

	router.Mount("/api", apiRouter)

	return router
}
