// Package actions holds the handlers behind the account pages.
package actions

import (
	"context"
	"errors"
	"net/http"

	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/web"
	"passenger-go/utils"
	"passenger-go/validations"
)

// msgDuplicate refuses an account whose platform and identifier another has.
const msgDuplicate = "An account with this platform and identifier already exists."

// Create stores the new-account form's submission.
func Create(d *web.Deps) *collage.Action {
	return collage.NewAction("account-create").
		WithMethods(http.MethodPost).
		WithHandler(createHandler(d)).
		Build()
}

// createHandler refuses a form missing a field or naming an existing account,
// with what was typed kept, and otherwise opens the new account.
func createHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		v := utils.PostForm(rc)
		rec := validations.Account(v)
		if !v.Valid() {
			return validate.Refuse(rc, v, rc.Page), nil
		}
		s, err := web.RequireSessionFrom(rc)
		if err != nil {
			return nil, err
		}
		defer s.DEK.Wipe()
		a, err := d.Store.CreateAccount(ctx, s.DEK, rec)
		if errors.Is(err, store.ErrDuplicate) {
			return utils.Refuse(rc, v, "platform", msgDuplicate), nil
		}
		if err != nil {
			return nil, err
		}
		return collage.SeeOtherPage(rc, "account", map[string]string{"id": a.ID})
	}
}
