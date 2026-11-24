package validations

import validate "github.com/Elagoht/collage-validate"

// Generate checks the random-password form: its length. Empty, not a number
// and out of range all get the same message, which says what to type. The
// look-alike form sends a source instead and has nothing to check.
func Generate(v *validate.Validator) *validate.IntField {
	const msg = "Choose a length between 8 and 4096 characters."
	return v.Field("length").Required(msg).Int(msg).Range(8, 4096, msg)
}
