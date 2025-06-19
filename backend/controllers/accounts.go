package controllers

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/guards"
	"passenger-go/backend/pipes"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/backend/utilities/router"

	"github.com/go-chi/chi"
	"github.com/go-playground/validator/v10"
)

type AccountsController struct {
	validator      *validator.Validate
	service        *services.AccountsService
	accountsRouter *router.Router
}

func NewAccountsController() *AccountsController {
	return &AccountsController{
		validator:      pipes.GetValidator(),
		service:        services.NewAccountsService(),
		accountsRouter: router.NewRouter(chi.NewRouter()),
	}
}

func (controller *AccountsController) MountAccountsRouter(router *chi.Mux) {
	controller.accountsRouter.Mux().Use(guards.JWTGuard)

	controller.accountsRouter.Get("/", controller.GetAccounts)
	controller.accountsRouter.Get("/identifiers", controller.GetUniqueIdentifiers)
	controller.accountsRouter.Get("/{id}", controller.GetPassphrase)
	controller.accountsRouter.Post("/", controller.CreateAccount)
	controller.accountsRouter.Put("/{id}", controller.UpdateAccount)
	controller.accountsRouter.Delete("/{id}", controller.DeleteAccount)

	router.Mount("/accounts", controller.accountsRouter.Mux())
}

func (controller *AccountsController) GetAccounts(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	accounts, err := controller.service.GetAccounts()
	if err != nil {
		return err
	}

	return json.NewEncoder(writer).Encode(accounts)
}

func (controller *AccountsController) GetUniqueIdentifiers(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	identifiers, err := controller.service.GetUniqueIdentifiers()
	if err != nil {
		return err
	}

	return json.NewEncoder(writer).Encode(identifiers)
}

func (controller *AccountsController) GetPassphrase(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	id := chi.URLParam(request, "id")
	if id == "" {
		return schemas.NewAPIError(
			schemas.ErrInvalidRequest,
			"Account ID is required",
			nil,
		)
	}

	passphrase, err := controller.service.GetPassphrase(id)
	if err != nil {
		return err
	}

	return json.NewEncoder(writer).Encode(passphrase)
}

func (controller *AccountsController) CreateAccount(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	body := &schemas.RequestAccountsUpsert{}
	if err := json.NewDecoder(request.Body).Decode(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrInvalidRequest,
			"Cannot validate request body",
			err,
		)
	}

	account, err := controller.service.CreateAccount(body)
	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusCreated)
	return json.NewEncoder(writer).Encode(account)
}

func (controller *AccountsController) UpdateAccount(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	id := chi.URLParam(request, "id")
	if id == "" {
		return schemas.NewAPIError(
			schemas.ErrInvalidRequest,
			"Account ID is required",
			nil,
		)
	}

	body := &schemas.RequestAccountsUpsert{}
	if err := json.NewDecoder(request.Body).Decode(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(body); err != nil {
		return schemas.NewAPIError(
			schemas.ErrInvalidRequest,
			"Cannot validate request body",
			err,
		)
	}

	err := controller.service.UpdateAccount(id, body)
	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusNoContent)
	return nil
}

func (controller *AccountsController) DeleteAccount(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	id := chi.URLParam(request, "id")
	if id == "" {
		return schemas.NewAPIError(
			schemas.ErrInvalidRequest,
			"Account ID is required",
			nil,
		)
	}

	err := controller.service.DeleteAccount(id)
	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusNoContent)
	return nil
}
