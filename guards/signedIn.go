package guards

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/session"
	"passenger-go/internal/web"
)

// SignedIn lets through requests carrying a live session. Without a vault it
// sends the reader to /setup; without a session to /login.
//
// It is the elagoht/session plugin's RequireUserPage("login") with three
// changes the plugin cannot be told. It decides on the manager's session
// (session.FromContext, which middleware.VaultSession puts in the context only
// for a live handle), not on the handle under the plugin's UserKey: a cookie
// whose handle is dead is not cleared (see VaultSession), so its UserKey says
// nothing. The vault check comes first. And next is added only for a GET or
// HEAD of anything but the home page: the home page needs none, and another
// method's path is not one to return to, since after login the browser would
// GET it.
func SignedIn(d *web.Deps) collage.GuardFunc {
	return func(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
		if dec, err := toSetup(ctx, d); dec != nil || err != nil {
			return dec, err
		}
		if _, ok := session.FromContext(ctx); ok {
			return nil, nil
		}
		login, err := collage.URL(ctx, "login", nil)
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(login)
		if err != nil {
			return nil, err
		}
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.URL.RequestURI() != "/" {
			q := u.Query()
			q.Set("next", r.URL.RequestURI())
			u.RawQuery = q.Encode()
		}
		return seeOther(u.String()), nil
	}
}
