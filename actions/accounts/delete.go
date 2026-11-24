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

// Delete removes account {id} once the confirmation page is submitted.
func Delete(d *web.Deps) *collage.Action {
	return collage.NewAction("account-delete").
		WithMethods(http.MethodPost).
		WithHandler(deleteHandler(d)).
		Build()
}

// deleteHandler deletes the account and goes back to the list; an unknown one
// is a 404.
func deleteHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		id := rc.Param("id")
		err := d.Store.DeleteAccount(ctx, id)
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("account %q: %w", id, collage.ErrNotFound)
		}
		if err != nil {
			return nil, err
		}
		return collage.SeeOtherPage(rc, "accounts", nil)
	}
}
