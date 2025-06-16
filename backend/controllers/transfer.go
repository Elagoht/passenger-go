package controllers

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/guards"
	"passenger-go/backend/pipes"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/backend/utilities/importer"
	"passenger-go/backend/utilities/router"

	"github.com/go-chi/chi"
	"github.com/go-playground/validator/v10"
)

type TransferController struct {
	transferRouter *router.Router
	validator      *validator.Validate
	service        *services.TransferService
}

func NewTransferController() *TransferController {
	return &TransferController{
		transferRouter: router.NewRouter(chi.NewRouter()),
		validator:      pipes.GetValidator(),
		service:        services.NewTransferService(),
	}
}

func (controller *TransferController) MountTransferRouter(router *chi.Mux) {
	controller.transferRouter.Mux().Use(guards.JWTGuard)

	controller.transferRouter.Post("/import", controller.Import)
	controller.transferRouter.Post("/export", controller.Export)

	router.Mount("/transfer", controller.transferRouter.Mux())
}

/*
A csv file is required.

Supported platforms:
- Firefox (UTF-8 CRLF double quotes comma separated)
- Chromium (UTF-8 LF no quotes comma separated)
*/
func (controller *TransferController) Import(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	file, _, err := request.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()

	csvFile, _, err := request.FormFile("file")
	if err != nil {
		return err
	}
	defer csvFile.Close()

	// First read to detect platform
	platform := importer.GetPlatform(csvFile)
	if platform.Fields == nil {
		return schemas.NewAPIError(
			schemas.ErrInvalidPlatform,
			"The uploaded file format is not supported. Please use Firefox or Chromium export format.",
			nil,
		)
	}

	// Reset file pointer for parsing
	if _, err := csvFile.Seek(0, 0); err != nil {
		return schemas.NewAPIError(
			schemas.ErrUnexpected,
			"Failed to process the file",
			err,
		)
	}

	accounts, err := platform.Parse(csvFile)
	if err != nil {
		return err
	}

	if len(accounts) == 0 {
		return schemas.NewAPIError(
			schemas.ErrUnprocessableEntity,
			"No accounts found in the CSV file",
			nil,
		)
	}

	importResult, err := controller.service.Import(accounts)
	if err != nil {
		return err
	}

	json.NewEncoder(writer).Encode(importResult)

	return nil
}

func (controller *TransferController) Export(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	csv, err := controller.service.Export()
	if err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "text/csv")
	writer.Header().Set("Content-Disposition", "attachment; filename=passenger-accounts.csv")
	writer.Write([]byte(csv))

	return nil
}
