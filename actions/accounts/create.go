package actions

import (
	"context"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// Create stores the new-account form's submission.
func Create(d *web.Deps) *collage.Action {
	return collage.NewAction("account-create").
		WithMethods(http.MethodPost).
		WithHandler(func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			f := readForm(rc.Request)
			if msg := problem(f); msg != "" {
				return refuse(rc, f, msg), nil
			}
			s, err := web.RequireSessionFrom(rc)
			if err != nil {
				return nil, err
			}
			defer s.DEK.Wipe()
			a, err := d.Store.CreateAccount(ctx, s.DEK, record(f))
			if isDuplicate(err) {
				return refuse(rc, f, msgDuplicate), nil
			}
			if err != nil {
				return nil, err
			}
			return collage.SeeOtherPage(rc, "account", map[string]string{"id": a.ID})
		}).
		Build()
}
