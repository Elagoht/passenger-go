package controllers

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/schemas"
	"passenger-go/backend/services"
	"passenger-go/backend/utilities"
	"passenger-go/backend/utilities/importer"

	"github.com/go-chi/chi"
	"github.com/go-playground/validator/v10"
)

type TransferController struct {
	transferRouter *utilities.Router
	validator      *validator.Validate
	service        *services.TransferService
}

func NewTransferController() *TransferController {
	return &TransferController{
		transferRouter: utilities.NewRouter(chi.NewRouter()),
	}
}

func (controller *TransferController) MountTransferRouter(router *chi.Mux) {
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

	json.NewEncoder(writer).Encode(accounts)
	return nil
}

func (controller *TransferController) Export(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	return nil
}
