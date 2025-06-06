package schemas

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Stack   error  `json:"-"`
}

func (error *APIError) Error() string {
	if error.Message != "" {
		return error.Message
	}
	return error.Code
}

func NewAPIError(code APIErrorCode, message string, stack error) *APIError {
	return &APIError{Code: string(code), Message: message, Stack: stack}
}

type APIErrorCode string

const (
	ErrDatabase                 APIErrorCode = "DATABASE_FAULT"
	ErrAlreadyInitialized       APIErrorCode = "ALREADY_INITIALIZED"
	ErrNotInitializedYet        APIErrorCode = "NOT_INITIALIZED_YET"
	ErrEncryptionFailed         APIErrorCode = "ENCRYPTION_FAILED"
	ErrRecoveryGenerationFailed APIErrorCode = "RECOVERY_KEY_GENERATION_FAILED"
	ErrInvalidCredentials       APIErrorCode = "INVALID_CREDENTIALS"
	ErrJWTGenerationFailed      APIErrorCode = "JWT_GENERATION_FAILED"
	ErrUnexpected               APIErrorCode = "UNEXPECTED"
)
