package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

type recoverView struct {
	Error  string
	NewKey string
}

// Recover resets the master passphrase with the recovery key and, once it
// has, shows the replacement recovery key.
func Recover(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("recover", "pages/auth/recover.html").
		WithTitle("Recover - Passenger").
		WithGuard(d.RequireGuest).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (recoverView, error) {
			key, _ := NewRecoveryKey.Get(rc)
			return recoverView{Error: formError(rc), NewKey: key}, nil
		})).
		Build()
}
