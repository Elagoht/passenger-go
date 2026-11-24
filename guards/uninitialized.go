// Package guards decides who may reach the pages and actions: signed-in
// readers, guests, and the first-run setup.
package guards

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// Uninitialized lets through requests only while no vault exists; afterwards
// it sends the reader to login.
func Uninitialized(d *web.Deps) collage.GuardFunc {
	return func(ctx context.Context, _ *http.Request) (*collage.GuardDecision, error) {
		ok, err := d.Store.Initialized(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			return seeOther("/login"), nil
		}
		return nil, nil
	}
}

// toSetup sends every request to /setup while no vault exists, and is nil
// once one does. The session plugin's guards know nothing of a vault.
func toSetup(ctx context.Context, d *web.Deps) (*collage.GuardDecision, error) {
	ok, err := d.Store.Initialized(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return seeOther("/setup"), nil
	}
	return nil, nil
}

// seeOther is a 303 to location. Guard targets are literal paths: a literal
// cannot fail at request time.
func seeOther(location string) *collage.GuardDecision {
	return &collage.GuardDecision{Status: http.StatusSeeOther, Location: location}
}
