package controllers

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"

	"github.com/go-chi/chi"
)

type AuthController struct {
	authRouter  *chi.Mux
	authService *services.AuthService
}

func NewAuthController() *AuthController {
	return &AuthController{
		authService: services.NewAuthService(),
	}
}

func (controller *AuthController) MountAuthRouter(router *chi.Mux) {
	controller.authRouter = chi.NewRouter()
	controller.authRouter.Get("/status", controller.Status)
	controller.authRouter.Post("/register", controller.RegisterUser)
	controller.authRouter.Post("/validate", controller.CompleteRegistration)
	controller.authRouter.Post("/login", controller.LoginUser)
	controller.authRouter.Post("/update", controller.UpdatePassphrase)

	router.Mount("/auth", controller.authRouter)
}

func (controller *AuthController) Status(
	writer http.ResponseWriter,
	request *http.Request,
) {
	initialized, err := controller.authService.Status()

	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(writer).Encode(schemas.ResponseIsInitialized{
		Initialized: initialized,
	})
}

func (controller *AuthController) RegisterUser(
	writer http.ResponseWriter,
	request *http.Request,
) {
	var body schemas.RequestAuthRegister
	json.NewDecoder(request.Body).Decode(&body)

	recovery, err := controller.authService.RegisterUser(body.Passphrase)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(writer).Encode(schemas.ResponseError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: err.Error(),
		})
		return
	}

	json.NewEncoder(writer).Encode(schemas.ResponseAuthRegister{
		Recovery: recovery,
	})
}

func (controller *AuthController) CompleteRegistration(
	writer http.ResponseWriter,
	request *http.Request,
) {
	var body schemas.RequestAuthCompleteRegistration
	json.NewDecoder(request.Body).Decode(&body)

	err := controller.authService.CompleteRegistration(body.Recovery)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(writer).Encode(schemas.ResponseError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: err.Error(),
		})
		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (controller *AuthController) LoginUser(
	writer http.ResponseWriter,
	request *http.Request,
) {
	var body schemas.RequestAuthLogin
	json.NewDecoder(request.Body).Decode(&body)

	token, err := controller.authService.LoginUser(body.Passphrase)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(writer).Encode(schemas.ResponseAuthLogin{
		Token: token,
	})
}

func (controller *AuthController) UpdatePassphrase(
	writer http.ResponseWriter,
	request *http.Request,
) {
	var body schemas.RequestAuthUpdatePassphrase
	json.NewDecoder(request.Body).Decode(&body)
}
