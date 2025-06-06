package schemas

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
	Stack   error  `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func NewAPIError(code APIErrorCode, message string, stack error) *APIError {
	return &APIError{Code: string(code), Message: message, Stack: stack}
}

type APIErrorCode string

const (
	ErrDatabase           APIErrorCode = "DATABASE_ERROR"
	ErrAlreadyInitialized APIErrorCode = "ALREADY_INITIALIZED"
)
