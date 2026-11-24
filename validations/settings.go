package validations

import validate "github.com/Elagoht/collage-validate"

// Passphrase checks the passphrase change before the current passphrase is:
// the new one and its repeat.
func Passphrase(v *validate.Validator) { newPassphrase(v) }
