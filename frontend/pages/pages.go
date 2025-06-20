package pages

import (
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/frontend/utilities/template"

	"github.com/go-chi/chi"
)

type PagesController struct {
	template        *template.TemplateManager
	authService     *services.AuthService
	accountsService *services.AccountsService
}

func NewPagesController() *PagesController {
	return &PagesController{
		template:        template.NewTemplateManager(),
		authService:     services.NewAuthService(),
		accountsService: services.NewAccountsService(),
	}
}

func (controller *PagesController) RouteApp(
	writer http.ResponseWriter,
	request *http.Request,
) {
	accounts, err := controller.accountsService.GetAccounts()
	if err != nil {
		accounts = []*schemas.ResponseAccount{}
	}

	controller.template.Render(writer, "app", "main", map[string]any{
		"Accounts": accounts,
		"Empty":    len(accounts) == 0,
		"Token":    request.CookiesNamed("token")[0].Value,
	})
}

func (controller *PagesController) RouteAccountDetails(
	writer http.ResponseWriter,
	request *http.Request,
) {
	id := chi.URLParam(request, "id")
	account, err := controller.accountsService.GetAccount(id)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	identifiers, err := controller.accountsService.GetUniqueIdentifiers()
	if err != nil {
		identifiers = []string{}
	}

	controller.template.Render(writer, "app", "details", map[string]any{
		"Account":     account,
		"Identifiers": identifiers,
	})
}

func (controller *PagesController) RouteAccountCreate(
	writer http.ResponseWriter,
	request *http.Request,
) {
	identifiers, err := controller.accountsService.GetUniqueIdentifiers()
	if err != nil {
		identifiers = []string{}
	}

	controller.template.Render(writer, "app", "create", map[string]any{
		"Identifiers": identifiers,
	})
}

func (controller *PagesController) RouteImport(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "app", "import", nil)
}

func (controller *PagesController) RouteExport(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "app", "export", nil)
}

func (controller *PagesController) RouteChangePassword(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "app", "change-password", nil)
}

func (controller *PagesController) RouteApiDocs(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "app", "api-docs", nil)
}

func (controller *PagesController) RouteLogin(
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

func (controller *PagesController) RouteRegister(
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

func (controller *PagesController) RouteRecover(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "auth", "recover", nil)
}
