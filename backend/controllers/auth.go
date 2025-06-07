package controllers

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/pipes"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/backend/utilities"

	"github.com/go-chi/chi"
	"github.com/go-playground/validator/v10"
)

type AuthController struct {
	authRouter  *utilities.Router
	authService *services.AuthService
	validator   *validator.Validate
}

func NewAuthController() *AuthController {
	return &AuthController{
		validator:   pipes.GetValidator(),
		authService: services.NewAuthService(),
		authRouter:  utilities.NewRouter(chi.NewRouter()),
	}
}

func (controller *AuthController) MountAuthRouter(router *chi.Mux) {
	controller.authRouter.Get("/status", controller.Status)
	controller.authRouter.Post("/register", controller.RegisterUser)
	controller.authRouter.Post("/validate", controller.CompleteRegistration)
	controller.authRouter.Post("/login", controller.LoginUser)
	controller.authRouter.Post("/update", controller.UpdatePassphrase)

	router.Mount("/auth", controller.authRouter.Mux())
}

func (controller *AuthController) Status(
	writer http.ResponseWriter,
	request *http.Request,
) (err error) {
	initialized, err := controller.authService.Status()

	if err != nil {
		return err
	}

	json.NewEncoder(writer).Encode(schemas.ResponseIsInitialized{
		Initialized: initialized,
	})

	return nil
}

func (controller *AuthController) RegisterUser(
	writer http.ResponseWriter,
	request *http.Request,
) (err error) {
	body := &schemas.RequestAuthRegister{}
	if err := json.NewDecoder(request.Body).Decode(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Cannot validate request body",
			err,
		)
	}

	recovery, err := controller.authService.RegisterUser(body.Passphrase)
	if err != nil {
		return err
	}

	json.NewEncoder(writer).Encode(schemas.ResponseAuthRegister{
		Recovery: recovery,
	})

	return nil
}

func (controller *AuthController) CompleteRegistration(
	writer http.ResponseWriter,
	request *http.Request,
) (err error) {
	body := &schemas.RequestAuthCompleteRegistration{}
	if err := json.NewDecoder(request.Body).Decode(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Cannot validate request body",
			err,
		)
	}

	err = controller.authService.CompleteRegistration(body.Recovery)
	if err != nil {
		return err
	}

	return nil
}

func (controller *AuthController) LoginUser(
	writer http.ResponseWriter,
	request *http.Request,
) (err error) {
	body := &schemas.RequestAuthLogin{}
	if err := json.NewDecoder(request.Body).Decode(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Cannot validate request body",
			err,
		)
	}

	token, err := controller.authService.LoginUser(body.Passphrase)
	if err != nil {
		return err
	}

	json.NewEncoder(writer).Encode(schemas.ResponseAuthLogin{
		Token: token,
	})

	return nil
}

func (controller *AuthController) UpdatePassphrase(
	writer http.ResponseWriter,
	request *http.Request,
) (err error) {
	body := &schemas.RequestAuthUpdatePassphrase{}
	if err := json.NewDecoder(request.Body).Decode(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Cannot validate request body",
			err,
		)
	}

	err = controller.authService.UpdatePassphrase(body.Passphrase)
	if err != nil {
		return err
	}

	return nil
}
