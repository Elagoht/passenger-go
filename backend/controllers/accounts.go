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

type AccountsController struct {
	validator      *validator.Validate
	service        *services.AccountsService
	accountsRouter *utilities.Router
}

func NewAccountsController() *AccountsController {
	return &AccountsController{
		validator:      pipes.GetValidator(),
		service:        services.NewAccountsService(),
		accountsRouter: utilities.NewRouter(chi.NewRouter()),
	}
}

func (controller *AccountsController) MountAccountsRouter(router *chi.Mux) {
	controller.accountsRouter.Post("/", controller.CreateAccount)
	controller.accountsRouter.Get("/", controller.GetAccountCards)
	controller.accountsRouter.Get("/{id}", controller.GetAccountDetails)
	controller.accountsRouter.Delete("/{id}", controller.DeleteAccount)

	router.Mount("/accounts", controller.accountsRouter.Mux())
}

func (controller *AccountsController) CreateAccount(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	account := &schemas.RequestAccountsCreate{}
	if err := json.NewDecoder(request.Body).Decode(account); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(account); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Cannot validate request body",
			err,
		)
	}

	createdAccountId, err := controller.service.CreateAccount(account)
	if err != nil {
		return err
	}

	json.NewEncoder(writer).Encode(
		schemas.ResponseAccountsCreate{
			Id: createdAccountId,
		},
	)
	writer.WriteHeader(http.StatusCreated)

	return nil
}

func (controller *AccountsController) UpdateAccount(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	id := chi.URLParam(request, "id")

	account := &schemas.RequestAccountsUpdate{}
	if err := json.NewDecoder(request.Body).Decode(account); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Invalid request body",
			err,
		)
	}

	if err := controller.validator.Struct(account); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"Cannot validate request body",
			err,
		)
	}

	err := controller.service.UpdateAccount(id, account)
	if err != nil {
		return err
	}

	return nil
}

func (controller *AccountsController) GetAccountCards(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	pagination, err := utilities.PaginationParams(request)
	if err != nil {
		return err
	}

	accountCards, err := controller.service.GetAccountCards(
		pagination.Page,
		pagination.Take,
	)
	if err != nil {
		return err
	}

	response := schemas.ResponseAccountCardList{
		Accounts: make([]schemas.ResponseAccountCard, len(accountCards)),
	}

	for i, account := range accountCards {
		response.Accounts[i] = schemas.ToResponseAccountCard(account)
	}

	json.NewEncoder(writer).Encode(response)

	return nil
}

func (controller *AccountsController) GetAccountDetails(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	id := chi.URLParam(request, "id")

	account, err := controller.service.GetAccountDetails(id)
	if err != nil {
		return err
	}

	json.NewEncoder(writer).Encode(account)

	return nil
}

func (controller *AccountsController) DeleteAccount(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	id := chi.URLParam(request, "id")

	err := controller.service.DeleteAccount(id)
	if err != nil {
		return err
	}

	writer.WriteHeader(http.StatusNoContent)

	return nil
}
