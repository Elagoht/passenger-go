package api_error

import (
	"net/http"
	"os"
	"passenger-go/backend/errors"
	"passenger-go/backend/schemas"
	"passenger-go/backend/utilities/logger"

	"github.com/joho/godotenv"
)

var isDev = false

func init() {
	godotenv.Load()

	isDev = os.Getenv("MODE") == "development"
}

var log = logger.GetLogger()

func HandleAPIError(writer http.ResponseWriter, err error) {
	apiError, ok := err.(*schemas.APIError)

	if err != nil {
		if isDev {
			if ok {
				log.Printf("API Error Code: %v", apiError.Code)
				log.Printf("API Error Message: %v", apiError.Message)
				if apiError.Stack != nil {
					log.Printf("API Error Stack: %v", apiError.Stack.Error())
				}
			} else {
				log.Printf("API Error: %v", err)
			}
		}

		if ok {
			errors.WriteHTTPError(writer, apiError)
		} else {
			errors.WriteHTTPError(writer, schemas.NewAPIError(schemas.ErrUnexpected, err.Error(), err))
		}
	}
}
