package errors

import (
	"encoding/json"
	"net/http"
	"passenger-go/backend/schemas"
)

var httpErrorMapping = map[schemas.APIErrorCode]int{
	schemas.ErrDatabase:                 500,
	schemas.ErrAlreadyInitialized:       409,
	schemas.ErrNotInitializedYet:        403,
	schemas.ErrEncryptionFailed:         500,
	schemas.ErrRecoveryGenerationFailed: 500,
	schemas.ErrInvalidCredentials:       401,
	schemas.ErrJWTGenerationFailed:      500,
}

func WriteHTTPError(writer http.ResponseWriter, error *schemas.APIError) {
	code, ok := httpErrorMapping[schemas.APIErrorCode(error.Code)]
	if !ok {
		code = 500
	}

	writer.WriteHeader(code)
	json.NewEncoder(writer).Encode(error)
}
