package frontend

import (
	"html/template"
	"net/http"
	"passenger-go/backend/services"
	"passenger-go/frontend/utilities/form"

	"github.com/go-chi/chi"
)

type FrontendController struct {
	templates   *template.Template
	router      *chi.Mux
	authService *services.AuthService
}

func NewFrontendController() (*FrontendController, error) {
	// Parse all templates
	templates, err := template.ParseGlob("frontend/templates/**/*.go.tmpl")
	if err != nil {
		return nil, err
	}

	return &FrontendController{
		templates:   templates,
		router:      chi.NewRouter(),
		authService: services.NewAuthService(),
	}, nil
}

func (c *FrontendController) MountFrontendRouter(router *chi.Mux) {
	// Serve static files
	fileServer := http.FileServer(http.Dir("frontend/static"))
	router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Mount frontend routes
	router.Get("/", c.handleIndex)
	router.Get("/login", c.routeLogin)
	router.Get("/register", c.routeRegister)

	// Form handlers
	router.Post("/register", c.formRegister)
}

func (controller *FrontendController) handleIndex(
	writer http.ResponseWriter,
	request *http.Request,
) {
	err := controller.templates.ExecuteTemplate(
		writer,
		"base",
		nil,
	)
	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusInternalServerError,
		)
	}
}

func (controller *FrontendController) routeLogin(
	writer http.ResponseWriter,
	request *http.Request,
) {
	err := controller.templates.ExecuteTemplate(
		writer,
		"pages/login.go.tmpl",
		nil,
	)
	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusInternalServerError,
		)
	}
}

func (controller *FrontendController) routeRegister(
	writer http.ResponseWriter,
	request *http.Request,
) {
	err := controller.templates.ExecuteTemplate(
		writer,
		"pages/register.go.tmpl",
		nil,
	)
	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusInternalServerError,
		)
	}
}

func (controller *FrontendController) formRegister(
	writer http.ResponseWriter,
	request *http.Request,
) {
	passphrase := request.FormValue("passphrase")
	confirmPassphrase := request.FormValue("confirm-passphrase")

	formError := form.ValidateRegisterForm(passphrase, confirmPassphrase)

	if formError != "" {
		err := controller.templates.ExecuteTemplate(
			writer,
			"pages/register.go.tmpl",
			map[string]string{
				"Error": formError,
			},
		)
		if err != nil {
			http.Error(
				writer,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}

	recoveryKey, err := controller.authService.RegisterUser(passphrase)
	if err != nil {
		http.Error(
			writer,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}

	// TODO: Save recovery key to local storage
	writer.Write([]byte(recoveryKey))
}
