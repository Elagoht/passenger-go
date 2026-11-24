package guards

import (
	"context"
	"net/http"

	session "github.com/Elagoht/collage-session"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
	"passenger-go/utils"
)

// Guest lets through readers with no session on an initialized vault: the
// elagoht/session plugin's RequireGuestPage("accounts"), which sends a
// signed-in reader to the next they carry or home, after the vault check the
// plugin cannot make. The plugin checks next with collage.SafeRedirect alone;
// utils.SafeRedirect also refuses one that only decodes to a local path.
func Guest(d *web.Deps) collage.GuardFunc {
	guest := session.RequireGuestPage("accounts")
	return func(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
		if dec, err := toSetup(ctx, d); dec != nil || err != nil {
			return dec, err
		}
		dec, err := guest(ctx, r)
		if dec != nil && err == nil {
			dec.Location = utils.SafeRedirect(dec.Location, "/")
		}
		return dec, err
	}
}
