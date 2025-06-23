package backend

import (
	"passenger-go/backend/controllers"

	"github.com/go-chi/chi"
)

var authController = controllers.NewAuthController()
var accountsController = controllers.NewAccountsController()
var transferController = controllers.NewTransferController()
var generateController = controllers.NewGenerateController()

func MountBackend(router *chi.Mux) *chi.Mux {
	apiRouter := chi.NewRouter()

	authController.MountAuthRouter(apiRouter)
	accountsController.MountAccountsRouter(apiRouter)
	transferController.MountTransferRouter(apiRouter)
	generateController.MountGenerateRouter(apiRouter)

	router.Mount("/api", apiRouter)

	return router
}
