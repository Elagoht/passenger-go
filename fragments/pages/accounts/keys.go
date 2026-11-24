// Package fragments holds the content of the account pages.
package fragments

import "github.com/Elagoht/collage/pkg/collage"

// Form is what an account form was submitted with, put back on the page an
// action refuses with.
type Form struct {
	Platform   string
	Identifier string
	Passphrase string
	URL        string
	Notes      string
}

// Submitted carries the refused submission to the form it came from.
var Submitted = collage.NewKey[Form]("accounts-submitted")

// FormError is the reason an action refused a submission.
var FormError = collage.NewKey[string]("accounts-form-error")

func formError(rc *collage.RenderContext) string {
	msg, _ := FormError.Get(rc)
	return msg
}
