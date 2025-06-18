package form

var registerErrors = map[string]string{
	"passphrase": "Passphrase is required",
	"confirm":    "Confirm passphrase is required",
	"match":      "Passphrases do not match",
	"length":     "Passphrase must be at least 12 characters long",
	"unknown":    "An unknown error occurred",
}

func ValidateRegisterForm(passphrase string, confirmPassphrase string) string {
	if passphrase == "" {
		return registerErrors["passphrase"]
	}
	if confirmPassphrase == "" {
		return registerErrors["confirm"]
	}

	if len(passphrase) < 12 {
		return registerErrors["length"]
	}

	if passphrase != confirmPassphrase {
		return registerErrors["match"]
	}

	return ""
}

var changePasswordErrors = map[string]string{
	"passphrase": "Passphrase is required",
	"confirm":    "Confirm passphrase is required",
	"match":      "Passphrases do not match",
	"length":     "Passphrase must be at least 12 characters long",
	"unknown":    "An unknown error occurred",
}

func ValidateChangePasswordForm(passphrase string, confirmPassphrase string) string {
	if passphrase == "" {
		return changePasswordErrors["passphrase"]
	}
	if confirmPassphrase == "" {
		return changePasswordErrors["confirm"]
	}

	if len(passphrase) < 12 {
		return changePasswordErrors["length"]
	}

	if passphrase != confirmPassphrase {
		return changePasswordErrors["match"]
	}

	return ""
}
