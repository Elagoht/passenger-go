package pipes

import (
	"github.com/go-playground/validator/v10"
)

func createValidator() *validator.Validate {
	validate := validator.New()
	return validate
}

var defaultValidator = createValidator()

func GetValidator() *validator.Validate {
	return defaultValidator
}
