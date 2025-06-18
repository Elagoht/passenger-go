package frontend

import (
	"net/http"
	"passenger-go/frontend/forms"
	"passenger-go/frontend/pages"
	"passenger-go/frontend/utilities/auth"
	"passenger-go/frontend/utilities/template"

	"github.com/go-chi/chi"
)

type FrontendController struct {
	template        *template.TemplateManager
	pagesController *pages.PagesController
	formsController *forms.FormsController
}

func NewFrontendController() (*FrontendController, error) {
	return &FrontendController{
		template:        template.NewTemplateManager(),
		pagesController: pages.NewPagesController(),
		formsController: forms.NewFormsController(),
	}, nil
}

func (controller *FrontendController) MountFrontendRouter(router *chi.Mux) {
	// Serve static files
	fileServer := http.FileServer(http.Dir("frontend/static"))
	router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Public routes
	router.Group(func(router chi.Router) {
		router.Use(auth.PublicMiddleware)

		router.Get("/login", controller.pagesController.RouteLogin)
		router.Get("/register", controller.pagesController.RouteRegister)
		router.Get("/recover", controller.pagesController.RouteRecover)

		router.Post("/register", controller.formsController.FormRegister)
		router.Post("/check", controller.formsController.FormCheck)
		router.Post("/complete", controller.formsController.FormComplete)
		router.Post("/login", controller.formsController.FormLogin)
		router.Post("/recover", controller.formsController.FormRecover)
	})

	// Protected routes
	router.Group(func(router chi.Router) {
		router.Use(auth.PrivateMiddleware)
		router.Get("/", controller.pagesController.RouteApp)
		router.Get("/accounts/{id}", controller.pagesController.RouteAccountDetails)
		router.Get("/create", controller.pagesController.RouteAccountCreate)
		router.Get("/import", controller.pagesController.RouteImport)
		router.Get("/export", controller.pagesController.RouteExport)
		router.Get("/change-password", controller.pagesController.RouteChangePassword)

		router.Post("/accounts/{id}", controller.formsController.FormAccountDetails)
		router.Post("/create", controller.formsController.FormAccountCreate)
		router.Post("/import", controller.formsController.FormImport)
		router.Post("/change-password", controller.formsController.FormChangePassword)
	})
}
