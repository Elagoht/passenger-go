package guards

import (
	"context"
	"net/http"

	session "github.com/Elagoht/collage-session"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// Guest lets through readers with no session on an initialized vault: the
// elagoht/session plugin's RequireGuestPage("accounts"), which sends a
// signed-in reader to the next they carry (when collage.SafeRedirect finds it
// a path on this site) or home, after the vault check the plugin cannot make.
func Guest(d *web.Deps) collage.GuardFunc {
	guest := session.RequireGuestPage("accounts")
	return func(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
		if dec, err := toSetup(ctx, d); dec != nil || err != nil {
			return dec, err
		}
		return guest(ctx, r)
	}
}
