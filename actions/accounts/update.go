package actions

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/web"
)

// Update saves the detail form's submission over account {id}.
func Update(d *web.Deps) *collage.Action {
	return collage.NewAction("account-update").
		WithMethods(http.MethodPost).
		WithHandler(func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			id := rc.Param("id")
			s, err := web.RequireSessionFrom(rc)
			if err != nil {
				return nil, err
			}
			defer s.DEK.Wipe()
			if _, err := d.Store.GetAccount(ctx, s.DEK, id); errors.Is(err, store.ErrNotFound) {
				return nil, fmt.Errorf("account %q: %w", id, collage.ErrNotFound)
			} else if err != nil {
				return nil, err
			}
			f := readForm(rc.Request)
			if msg := problem(f); msg != "" {
				return refuse(rc, f, msg), nil
			}
			err = d.Store.UpdateAccount(ctx, s.DEK, id, record(f))
			if isDuplicate(err) {
				return refuse(rc, f, msgDuplicate), nil
			}
			if errors.Is(err, store.ErrNotFound) {
				return nil, fmt.Errorf("account %q: %w", id, collage.ErrNotFound)
			}
			if err != nil {
				return nil, err
			}
			return collage.SeeOtherPage(rc, "account", map[string]string{"id": id})
		}).
		Build()
}
