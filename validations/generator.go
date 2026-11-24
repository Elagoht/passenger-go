package validations

import validate "github.com/Elagoht/collage-validate"

// Generate checks the random-password form: its length. The look-alike form
// sends a source instead and has nothing to check.
func Generate(v *validate.Validator) *validate.IntField {
	const msg = "Length must be a number."
	return v.Field("length").Required(msg).Int(msg)
}
