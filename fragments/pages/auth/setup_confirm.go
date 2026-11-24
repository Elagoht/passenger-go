package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

type setupConfirmView struct {
	Recovery string
	Error    string
}

// SetupConfirm shows the pending setup's recovery key and asks for it back.
// Its guard turns away requests without a live pending setup.
func SetupConfirm(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("setup-confirm", "pages/auth/setup_confirm.html").
		WithTitle("Recovery key - Passenger").
		WithGuard(d.RequirePendingSetup).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (setupConfirmView, error) {
			_, recovery, dek, _ := d.Pending.Get(web.PendingToken(rc.Request))
			dek.Wipe()
			return setupConfirmView{Recovery: recovery, Error: formError(rc)}, nil
		})).
		Build()
}
