package guards

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
	"passenger-go/internal/web"
	"passenger-go/utils"
)

// Guest lets through readers with no live session on an initialized vault,
// and sends a signed-in reader to the next they carry or home: the
// elagoht/session plugin's RequireGuestPage("accounts"), after the vault
// check the plugin cannot make.
//
// Like SignedIn it decides on the manager's session (session.FromContext), not
// on the plugin's UserKey: a cookie whose handle is dead keeps its UserKey
// (middleware.VaultSession no longer clears it), and the plugin's guard would
// send its reader home, where SignedIn sends them back to login, in a loop.
// The plugin checks next with collage.SafeRedirect alone; utils.SafeRedirect
// also refuses one that only decodes to a local path.
func Guest(d *web.Deps) collage.GuardFunc {
	return func(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
		if dec, err := toSetup(ctx, d); dec != nil || err != nil {
			return dec, err
		}
		if _, ok := session.FromContext(ctx); !ok {
			return nil, nil
		}
		home, err := collage.URL(ctx, "accounts", nil)
		if err != nil {
			return nil, err
		}
		return seeOther(utils.SafeRedirect(r.URL.Query().Get("next"), home)), nil
	}
}
