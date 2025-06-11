package errors

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/schemas"
)

var httpErrorMapping = map[schemas.APIErrorCode]int{
	schemas.ErrInvalidRequest:           400,
	schemas.ErrInvalidCredentials:       401,
	schemas.ErrNotInitializedYet:        403,
	schemas.ErrAccountNotFound:          404,
	schemas.ErrAlreadyInitialized:       409,
	schemas.ErrAccountAlreadyExists:     409,
	schemas.ErrAnotherAccountFound:      409,
	schemas.ErrUnprocessableEntity:      422,
	schemas.ErrEncryptionFailed:         500,
	schemas.ErrDecryptionFailed:         500,
	schemas.ErrRecoveryGenerationFailed: 500,
	schemas.ErrJWTGenerationFailed:      500,
	schemas.ErrUnexpected:               500,
	schemas.ErrDatabase:                 500,
	schemas.ErrInvalidPlatform:          400,
}

func WriteHTTPError(writer http.ResponseWriter, err error) {
	apiError, ok := err.(*schemas.APIError)
	if !ok {
		apiError = schemas.NewAPIError(
			schemas.ErrUnexpected,
			"An unknown error occurred",
			err,
		)
	}

	code, ok := httpErrorMapping[schemas.APIErrorCode(apiError.Code)]
	if !ok {
		code = 500
	}

	writer.WriteHeader(code)
	json.NewEncoder(writer).Encode(apiError)
}
