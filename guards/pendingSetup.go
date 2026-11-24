package guards

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// PendingSetup lets through requests while no vault exists and the request
// carries the token of a live pending setup; without one it sends the reader
// back to /setup.
func PendingSetup(d *web.Deps) collage.GuardFunc {
	uninitialized := Uninitialized(d)
	return func(ctx context.Context, r *http.Request) (*collage.GuardDecision, error) {
		if dec, err := uninitialized(ctx, r); dec != nil || err != nil {
			return dec, err
		}
		if !d.HasPendingSetup(r) {
			return seeOther("/setup"), nil
		}
		return nil, nil
	}
}
