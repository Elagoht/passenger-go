package validations

import (
	validate "github.com/Elagoht/collage-validate"

	"passenger-go/internal/vault"
)

// Account checks the account form, new or edited, and returns what it holds.
// Site, username and address are trimmed; the passphrase and the notes are
// taken as typed.
func Account(v *validate.Validator) vault.Record {
	platform := v.Field("platform").Required("Platform is required.")
	identifier := v.Field("identifier").Required("Identifier is required.")
	// Required trims to decide, so a passphrase of spaces is refused; it is
	// stored as typed.
	v.Field("passphrase").Required("Passphrase is required.")
	url := v.Field("url")
	return vault.Record{
		Platform:   platform.Value(),
		Identifier: identifier.Value(),
		Passphrase: v.Value("passphrase"),
		URL:        url.Value(),
		Notes:      v.Value("notes"),
	}
}
