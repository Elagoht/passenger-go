// Package fragments holds the content of the setup, login and recovery pages.
package fragments

import "github.com/Elagoht/collage/pkg/collage"

// FormError is the message an action puts on the page it re-renders: a
// refused submission (422) or a client backing off (429).
var FormError = collage.NewKey[string]("auth-form-error")

// NewRecoveryKey is the recovery key /recover just issued, shown once.
var NewRecoveryKey = collage.NewKey[string]("auth-new-recovery-key")

// formError reads FormError, empty when none was set.
func formError(rc *collage.RenderContext) string {
	msg, _ := FormError.Get(rc)
	return msg
}
