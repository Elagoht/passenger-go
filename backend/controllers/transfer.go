package controllers

import (
	"net/http"
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

	platform, err := importer.DeterminePlatform(csvFile)
	if err != nil {
		return err
	}
	writer.Write([]byte(platform))

	return nil
}

func (controller *TransferController) Export(
	writer http.ResponseWriter,
	request *http.Request,
) error {
	return nil
}
