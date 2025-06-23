package controllers

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/pipes"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/backend/utilities/router"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/go-playground/validator/v10"
)

type GenerateController struct {
	service   *services.GenerateService
	validator *validator.Validate
	router    *router.Router
}

func NewGenerateController() *GenerateController {
	return &GenerateController{
		service:   services.NewGenerateService(),
		validator: pipes.GetValidator(),
		router:    router.NewRouter(chi.NewRouter()),
	}
}

func (controller *GenerateController) MountGenerateRouter(router *chi.Mux) {
	controller.router.Get("/new", controller.GeneratePassphrase)
	controller.router.Post("/alternative", controller.AlternatePassphrase)

	router.Mount("/generate", controller.router.Mux())
}

func (controller *GenerateController) GeneratePassphrase(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	length := request.URL.Query().Get("length")

	if length == "" {
		length = "32"
	}

	lengthInt, err := strconv.Atoi(length)
	if err != nil {
		return schemas.NewAPIError(
			schemas.ErrInvalidLength,
			"Invalid length",
			err,
		)
	}

	json.NewEncoder(writer).Encode(
		controller.service.Generate(lengthInt),
	)

	return nil
}

func (controller *GenerateController) AlternatePassphrase(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	body := &schemas.RequestAlternate{}
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

	json.NewEncoder(writer).Encode(
		controller.service.Alternate(body.Passphrase),
	)

	return nil
}
