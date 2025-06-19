package forms

import (
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/backend/utilities/importer"
	"passenger-go/frontend/utilities/form"
	"passenger-go/frontend/utilities/template"

	"github.com/go-chi/chi"
)

type FormsController struct {
	template        *template.TemplateManager
	authService     *services.AuthService
	accountsService *services.AccountsService
	transferService *services.TransferService
}

func NewFormsController() *FormsController {
	return &FormsController{
		template:        template.NewTemplateManager(),
		authService:     services.NewAuthService(),
		accountsService: services.NewAccountsService(),
		transferService: services.NewTransferService(),
	}
}

func (controller *FormsController) FormRegister(
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

func (controller *FormsController) FormCheck(
	writer http.ResponseWriter,
	request *http.Request,
) {
	controller.template.Render(writer, "auth", "check", nil)
}

func (controller *FormsController) FormComplete(
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

func (controller *FormsController) FormLogin(
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

func (controller *FormsController) FormAccountDetails(
	writer http.ResponseWriter,
	request *http.Request,
) {
	id := chi.URLParam(request, "id")
	platform := request.FormValue("platform")
	identifier := request.FormValue("identifier")
	passphrase := request.FormValue("passphrase")
	url := request.FormValue("url")
	notes := request.FormValue("notes")

	err := controller.accountsService.UpdateAccount(id, &schemas.RequestAccountsUpsert{
		Platform:   platform,
		Identifier: identifier,
		Passphrase: passphrase,
		Url:        url,
		Notes:      notes,
	})
	if err != nil {
		controller.template.Render(writer, "app", "details", map[string]any{
			"Error": err.Error(),
			"Account": &schemas.ResponseAccountDetails{
				Id:         id,
				Platform:   platform,
				Identifier: identifier,
				Passphrase: passphrase,
				Url:        url,
				Notes:      notes,
			},
		})
		return
	}

	http.Redirect(writer, request, "/", http.StatusFound)
}

func (controller *FormsController) FormAccountCreate(
	writer http.ResponseWriter,
	request *http.Request,
) {
	platform := request.FormValue("platform")
	identifier := request.FormValue("identifier")
	passphrase := request.FormValue("passphrase")
	url := request.FormValue("url")
	notes := request.FormValue("notes")

	account, err := controller.accountsService.CreateAccount(&schemas.RequestAccountsUpsert{
		Platform:   platform,
		Identifier: identifier,
		Passphrase: passphrase,
		Url:        url,
		Notes:      notes,
	})

	if err != nil {
		controller.template.Render(writer, "app", "create", map[string]string{
			"Error": err.Error(),
		})
		return
	}

	controller.template.Render(writer, "app", "details", map[string]any{
		"Account": account,
		"Message": "Account created successfully",
	})
}

func (controller *FormsController) FormImport(
	writer http.ResponseWriter,
	request *http.Request,
) {

	file, _, err := request.FormFile("file")
	if err != nil {
		controller.template.Render(writer, "app", "import", map[string]string{
			"Error": err.Error(),
		})
		return
	}
	defer file.Close()

	csvFile, _, err := request.FormFile("file")
	if err != nil {
		controller.template.Render(writer, "app", "import", map[string]string{
			"Error": err.Error(),
		})
		return
	}
	defer csvFile.Close()

	// First read to detect platform
	platform := importer.GetPlatform(csvFile)
	if platform.Fields == nil {
		controller.template.Render(writer, "app", "import", map[string]string{
			"Error": "The uploaded file format is not supported. Please use Firefox or Chromium export format.",
		})
		return
	}

	// Reset file pointer for parsing
	if _, err := csvFile.Seek(0, 0); err != nil {
		controller.template.Render(writer, "app", "import", map[string]string{
			"Error": err.Error(),
		})
		return
	}

	accounts, err := platform.Parse(csvFile)
	if err != nil {
		controller.template.Render(writer, "app", "import", map[string]string{
			"Error": err.Error(),
		})
		return
	}

	if len(accounts) == 0 {
		controller.template.Render(writer, "app", "import", map[string]string{
			"Error": "No accounts found in the CSV file",
		})
		return
	}

	importResult, err := controller.transferService.Import(accounts)
	if err != nil {
		controller.template.Render(writer, "app", "import", map[string]string{
			"Error": err.Error(),
		})
		return
	}

	if len(importResult.FailedOnes) > 0 {
		controller.template.Render(writer, "app", "import", map[string]any{
			"SuccessCount": importResult.SuccessCount,
			"FailedOnes":   importResult.FailedOnes,
		})
		return
	}

	controller.template.Render(writer, "app", "import", map[string]any{
		"SuccessCount": importResult.SuccessCount,
	})
}

func (controller *FormsController) FormChangePassword(
	writer http.ResponseWriter,
	request *http.Request,
) {
	passphrase := request.FormValue("passphrase")
	confirmPassphrase := request.FormValue("confirmPassphrase")

	formError := form.ValidateChangePasswordForm(passphrase, confirmPassphrase)

	if formError != "" {
		controller.template.Render(writer, "app", "change-password", map[string]string{
			"Error": formError,
		})
		return
	}

	err := controller.authService.UpdatePassphrase(passphrase)
	if err != nil {
		controller.template.Render(writer, "app", "change-password", map[string]string{
			"Error": err.Error(),
		})
		return
	}

	controller.template.Render(writer, "app", "change-password", map[string]string{
		"Message": "Passphrase changed successfully",
	})
}

func (controller *FormsController) FormRecover(
	writer http.ResponseWriter,
	request *http.Request,
) {
	recoveryKey := request.FormValue("recoveryKey")
	newPassphrase := request.FormValue("newPassphrase")
	confirmNewPassphrase := request.FormValue("confirmNewPassphrase")

	formError := form.ValidateRecoverForm(recoveryKey, newPassphrase, confirmNewPassphrase)

	if formError != "" {
		controller.template.Render(writer, "auth", "recover", map[string]string{
			"Error": formError,
		})
		return
	}

	err := controller.authService.RecoverUser(recoveryKey, newPassphrase)
	if err != nil {
		controller.template.Render(writer, "auth", "recover", map[string]string{
			"Error": err.Error(),
		})
		return
	}

	controller.template.Render(writer, "auth", "recover", map[string]string{
		"Message": "Passphrase recovered successfully",
	})
}

func (controller *FormsController) FormLogout(
	writer http.ResponseWriter,
	request *http.Request,
) {
	// Clear the token cookie by setting it with MaxAge of -1
	http.SetCookie(writer, &http.Cookie{
		Name:   "token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Redirect to login page
	http.Redirect(writer, request, "/login", http.StatusFound)
}
