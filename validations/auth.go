package validations

import validate "github.com/Elagoht/collage-validate"

// Setup checks the setup form before its code is: the new master passphrase
// and its repeat. The setup code is checked after the backoff, by the action.
func Setup(v *validate.Validator) { newPassphrase(v) }

// Recover checks the recovery form before its key is: the new master
// passphrase and its repeat.
func Recover(v *validate.Validator) { newPassphrase(v) }
