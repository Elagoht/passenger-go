package guards

import (
	"context"
	"net/http"
	"net/url"

	session "github.com/Elagoht/collage-session"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// SignedIn lets through requests carrying a live session. Without a vault it
// sends the reader to /setup; without a session to /login.
//
// It is the elagoht/session plugin's RequireUserPage("login"), which reads the
// handle under session.UserKey (middleware.VaultSession has already cleared one
// whose DEK is gone), with two changes the plugin cannot be told: the vault
// check first, and next only for a GET or HEAD of anything but the home page.
// The plugin always adds next; the home page needs none, and another method's
// path is not one to return to: after login the browser would GET it.
func SignedIn(d *web.Deps) collage.GuardFunc {
	user := session.RequireUserPage("login")
	return func(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
		if dec, err := toSetup(ctx, d); dec != nil || err != nil {
			return dec, err
		}
		dec, err := user(ctx, r)
		if dec == nil || err != nil {
			return dec, err
		}
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) || r.URL.RequestURI() == "/" {
			u, err := url.Parse(dec.Location)
			if err != nil {
				return nil, err
			}
			q := u.Query()
			q.Del("next")
			u.RawQuery = q.Encode()
			dec.Location = u.String()
		}
		return dec, nil
	}
}
