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
	schemas.ErrAlreadyInitialized:       409,
	schemas.ErrAccountAlreadyExists:     409,
	schemas.ErrUnprocessableEntity:      422,
	schemas.ErrEncryptionFailed:         500,
	schemas.ErrRecoveryGenerationFailed: 500,
	schemas.ErrJWTGenerationFailed:      500,
	schemas.ErrUnexpected:               500,
	schemas.ErrDatabase:                 500,
	schemas.ErrInvalidPlatform:          400,
}

func WriteHTTPError(writer http.ResponseWriter, error *schemas.APIError) {
	code, ok := httpErrorMapping[schemas.APIErrorCode(error.Code)]
	if !ok {
		code = 500
	}

	writer.WriteHeader(code)
	json.NewEncoder(writer).Encode(error)
}
