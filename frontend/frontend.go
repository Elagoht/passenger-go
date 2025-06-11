package frontend

import (
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/frontend/utilities/auth"
	"passenger-go/frontend/utilities/form"
	"passenger-go/frontend/utilities/template"

	"github.com/go-chi/chi"
)

type FrontendController struct {
	template    *template.TemplateManager
	router      *chi.Mux
	authService *services.AuthService
}

func NewFrontendController() (*FrontendController, error) {
	return &FrontendController{
		template:    template.NewTemplateManager(),
		router:      chi.NewRouter(),
		authService: services.NewAuthService(),
	}, nil
}

func (controller *FrontendController) MountFrontendRouter(router *chi.Mux) {
	// Serve static files
	fileServer := http.FileServer(http.Dir("frontend/static"))
	router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Auth routes
	router.Get("/login", controller.routeLogin)
	router.Get("/register", controller.routeRegister)

	// Form handlers
	router.Post("/register", controller.formRegister)
	router.Post("/check", controller.formCheck)
	router.Post("/complete", controller.formComplete)
	router.Post("/login", controller.formLogin)

	// Protected routes
	router.Group(func(router chi.Router) {
		router.Use(auth.Middleware)
		router.Get("/", controller.routeApp)
	})
}

func (controller *FrontendController) routeApp(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "app", "main", nil)
}

func (controller *FrontendController) routeLogin(
	writer http.ResponseWriter,
	request *http.Request,
) {
	initialized, err := controller.authService.Status()
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	if !initialized {
		http.Redirect(writer, request, "/register", http.StatusSeeOther)
		return
	}

	controller.template.Render(writer, "auth", "login", nil)
}

func (controller *FrontendController) routeRegister(
	writer http.ResponseWriter,
	request *http.Request,
) {
	initialized, err := controller.authService.Status()
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	if initialized {
		http.Redirect(writer, request, "/login", http.StatusSeeOther)
		return
	}

	controller.template.Render(writer, "auth", "register", nil)
}

func (controller *FrontendController) formRegister(
	writer http.ResponseWriter,
	request *http.Request,
) {
	passphrase := request.FormValue("passphrase")
	confirmPassphrase := request.FormValue("confirm-passphrase")

	formError := form.ValidateRegisterForm(passphrase, confirmPassphrase)

	if formError != "" {
		controller.template.Render(writer, "auth", "register", map[string]string{
			"Error": formError,
		})
		return
	}

	recovery, err := controller.authService.RegisterUser(passphrase)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	controller.template.Render(writer, "auth", "validate", map[string]string{
		"Recovery": recovery,
	})
}

func (controller *FrontendController) formCheck(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "auth", "check", nil)
}

func (controller *FrontendController) formComplete(
	writer http.ResponseWriter,
	request *http.Request,
) {
	recovery := request.FormValue("recovery")

	err := controller.authService.CompleteRegistration(recovery)
	if err != nil {
		apiError, ok := err.(*schemas.APIError)
		if ok {
			if apiError.Code == string(schemas.ErrAlreadyInitialized) {
				http.Redirect(writer, request, "/login", http.StatusFound)
				return
			} else {
				controller.template.Render(writer, "auth", "complete", map[string]string{
					"Error":    apiError.Message,
					"Recovery": recovery,
				})
			}
		} else {
			controller.template.Render(writer, "auth", "complete", map[string]string{
				"Error": err.Error(),
			})
		}
		return
	}

	controller.template.Render(writer, "auth", "complete", nil)
}

func (controller *FrontendController) formLogin(
	writer http.ResponseWriter,
	request *http.Request,
) {
	passphrase := request.FormValue("passphrase")

	token, err := controller.authService.LoginUser(passphrase)
	if err != nil {
		controller.template.Render(writer, "auth", "login", map[string]string{
			"Error": err.Error(),
		})
		return
	}

	http.SetCookie(writer, &http.Cookie{
		Name:   "token",
		Value:  token,
		Path:   "/",
		MaxAge: 360,
	})

	http.Redirect(writer, request, "/", http.StatusFound)
}
