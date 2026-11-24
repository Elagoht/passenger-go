// Package layouts holds the page shells: App for the unlocked vault, Auth for
// the setup, login and recovery screens.
package layouts

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// appView is the App layout's data: when the session ends, in Unix ms, for
// static/session.js to read from <body data-session-expires>.
type appView struct {
	ExpiresAtMS int64
	// RemainingMS is the time left by the server's clock, which the script
	// counts down on its own monotonic clock.
	RemainingMS int64
}

// App is the shell of every page behind the vault: navigation, logout and the
// session-expiry banner. It is guarded with RequireSession, so every page
// wrapped in it, and every action on such a page, needs a live session.
func App(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("app-layout", "layouts/app.html").
		WithTitle("Passenger").
		WithGuard(d.RequireSession).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (appView, error) {
			s := web.SessionFrom(rc)
			if s == nil {
				return appView{}, nil
			}
			return appView{
				ExpiresAtMS: d.Sessions.ExpiresAt(s).UnixMilli(),
				RemainingMS: d.Sessions.Remaining(s).Milliseconds(),
			}, nil
		})).
		Build()
}
