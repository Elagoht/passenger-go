// Package validations holds each form's checks, one function per form.
package validations

import (
	"unicode/utf8"

	validate "github.com/Elagoht/collage-validate"
)

// MinPassphraseRunes is the shortest master passphrase accepted.
const MinPassphraseRunes = 12

// Messages for a new master passphrase.
const (
	msgPassphraseShort  = "The passphrase must be at least 12 characters."
	msgPassphraseDiffer = "The passphrases do not match."
)

// newPassphrase checks a new master passphrase and its repeat. The checks read
// the values as typed: validate trims what its own checks see, and a
// passphrase is wrapped as typed, so "pass " and "pass" must not count as the
// same passphrase, and spaces must not make one long enough.
func newPassphrase(v *validate.Validator) {
	pass, confirm := v.Value("passphrase"), v.Value("confirm")
	if utf8.RuneCountInString(pass) < MinPassphraseRunes {
		v.Fail("passphrase", msgPassphraseShort)
		return
	}
	if pass != confirm {
		v.Fail("confirm", msgPassphraseDiffer)
	}
}
