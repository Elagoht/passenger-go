package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

type setupView struct {
	Error string
}

// Setup is the first-run form: setup code, master passphrase and its repeat.
// It is reachable only while no vault exists.
func Setup(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("setup", "pages/auth/setup.html").
		WithTitle("Setup - Passenger").
		WithGuard(d.RequireUninitialized).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (setupView, error) {
			return setupView{Error: formError(rc)}, nil
		})).
		Build()
}
