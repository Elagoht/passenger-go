package utilities

import (
	"net/http"
	"os"
	"passenger-go/backend/errors"
	"passenger-go/backend/schemas"

	"github.com/joho/godotenv"
)

var isDev = false

func init() {
	godotenv.Load()

	isDev = os.Getenv("MODE") == "development"
}

func HandleAPIError(writer http.ResponseWriter, err error) {
	apiError, ok := err.(*schemas.APIError)

	if err != nil {
		if isDev {
			if ok {
				logger.Printf("API Error Code: %v", apiError.Code)
				logger.Printf("API Error Message: %v", apiError.Message)
				if apiError.Stack != nil {
					logger.Printf("API Error Stack: %v", apiError.Stack.Error())
				}
			} else {
				logger.Printf("API Error: %v", err)
			}
		}

		if ok {
			errors.WriteHTTPError(writer, apiError)
		} else {
			errors.WriteHTTPError(writer, schemas.NewAPIError(schemas.ErrUnexpected, err.Error(), err))
		}
	}
}
