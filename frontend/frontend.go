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

type TemplateData struct {
	Year int
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

func (c *FrontendController) handleIndex(w http.ResponseWriter, r *http.Request) {
	err := c.templates.ExecuteTemplate(w, "base", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *FrontendController) routeLogin(w http.ResponseWriter, r *http.Request) {
	err := c.templates.ExecuteTemplate(w, "pages/login.go.tmpl", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *FrontendController) routeRegister(w http.ResponseWriter, r *http.Request) {
	err := c.templates.ExecuteTemplate(w, "pages/register.go.tmpl", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (c *FrontendController) formRegister(w http.ResponseWriter, r *http.Request) {
	passphrase := r.FormValue("passphrase")
	confirmPassphrase := r.FormValue("confirm-passphrase")

	formError := form.ValidateRegisterForm(passphrase, confirmPassphrase)

	if formError != "" {
		err := c.templates.ExecuteTemplate(w, "pages/register.go.tmpl", map[string]string{
			"Error": formError,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	recoveryKey, err := c.authService.RegisterUser(passphrase)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: Save recovery key to local storage
	w.Write([]byte(recoveryKey))
}
