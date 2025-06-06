package utilities

import (
	"encoding/json"
	"net/http"
	"os"
	"passenger-go/backend/schemas"

	"github.com/joho/godotenv"
)

var isDev = false

func init() {
	godotenv.Load()

	isDev = os.Getenv("ENV") == "development"
}

func HandleAPIError(writer http.ResponseWriter, err error) {
	if err != nil {
		if isDev {
			apiError, ok := err.(*schemas.APIError)
			if ok {
				Logger.Printf("API Error Code: %v", apiError.Code)
				Logger.Printf("API Error Message: %v", apiError.Message)
				Logger.Printf("API Error Stack: %v", apiError.Stack.Error())
			} else {
				Logger.Printf("API Error: %v", err)
			}
		}

		json.NewEncoder(writer).Encode(schemas.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: err.Error(),
		})
	}
}
