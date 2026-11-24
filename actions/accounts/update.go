package actions

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/web"
	"passenger-go/utils"
	"passenger-go/validations"
)

// Update saves the detail form's submission over account {id}.
func Update(d *web.Deps) *collage.Action {
	return collage.NewAction("account-update").
		WithMethods(http.MethodPost).
		WithHandler(updateHandler(d)).
		Build()
}

// updateHandler answers an unknown account with 404 before looking at the
// form, then refuses it as Create does, or saves it and shows the account.
func updateHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
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
		v := validate.Form(rc)
		rec := validations.Account(v)
		if !v.Valid() {
			return validate.Refuse(rc, v, rc.Page), nil
		}
		err = d.Store.UpdateAccount(ctx, s.DEK, id, rec)
		if errors.Is(err, store.ErrDuplicate) {
			return utils.Refuse(rc, v, "platform", msgDuplicate), nil
		}
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("account %q: %w", id, collage.ErrNotFound)
		}
		if err != nil {
			return nil, err
		}
		return collage.SeeOtherPage(rc, "account", map[string]string{"id": id})
	}
}
