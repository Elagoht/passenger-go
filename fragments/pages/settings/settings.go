// Package fragments holds the content of the settings pages.
package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"
)

// FormError is why a settings action refused a submission.
var FormError = collage.NewKey[string]("settings-form-error")

// NewRecoveryKey is the recovery key just issued, shown once.
var NewRecoveryKey = collage.NewKey[string]("settings-new-recovery-key")

func formError(rc *collage.RenderContext) string {
	msg, _ := FormError.Get(rc)
	return msg
}

type passphraseView struct {
	Error string
	Saved bool
}

// Passphrase changes the master passphrase.
func Passphrase() *collage.Fragment {
	return collage.NewFragment("settings-passphrase", "pages/settings/passphrase.html").
		WithTitle("Settings - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (passphraseView, error) {
			return passphraseView{
				Error: formError(rc),
				Saved: rc.Request.URL.Query().Get("saved") == "1",
			}, nil
		})).
		Build()
}

type recoveryView struct {
	Error  string
	NewKey string
}

// Recovery issues a new recovery key.
func Recovery() *collage.Fragment {
	return collage.NewFragment("settings-recovery", "pages/settings/recovery.html").
		WithTitle("Recovery key - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (recoveryView, error) {
			key, _ := NewRecoveryKey.Get(rc)
			return recoveryView{Error: formError(rc), NewKey: key}, nil
		})).
		Build()
}
