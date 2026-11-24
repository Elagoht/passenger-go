package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

type loginView struct {
	Next  string
	Error string
}

// Login is the unlock form. It carries next through the post in a hidden
// field.
func Login(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("login", "pages/auth/login.html").
		WithTitle("Login - Passenger").
		WithGuard(d.RequireGuest).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (loginView, error) {
			r := rc.Request
			return loginView{
				Next:  web.SafeNext(r.FormValue("next")),
				Error: formError(rc),
			}, nil
		})).
		Build()
}
