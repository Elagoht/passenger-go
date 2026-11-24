package fragments

import (
	"context"
	"errors"
	"fmt"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/web"
)

type formView struct {
	ID         string // empty for a new account
	Platform   string
	Identifier string
	Passphrase string
	URL        string
	Notes      string
	Error      string
}

// New is the empty account form. Reachable only behind a session.
func New() *collage.Fragment {
	return collage.NewFragment("account-new", "pages/accounts/form.html").
		WithTitle("New account - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (formView, error) {
			return submittedView(rc, formView{}), nil
		})).
		Build()
}

// Detail shows one account for editing, its passphrase masked. An unknown id
// is a 404.
func Detail(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("account-detail", "pages/accounts/form.html").
		WithTitle("Account - Passenger").
		Required().
		WithData(collage.Load(func(ctx context.Context, rc *collage.RenderContext) (formView, error) {
			id := rc.Param("id")
			if _, refused := Submitted.Get(rc); refused {
				// The action already found the account; show what was typed.
				return submittedView(rc, formView{ID: id}), nil
			}
			a, err := load(ctx, d, rc, id)
			if err != nil {
				return formView{}, err
			}
			return formView{
				ID: a.ID, Platform: a.Platform, Identifier: a.Identifier,
				Passphrase: a.Passphrase, URL: a.URL, Notes: a.Notes,
			}, nil
		})).
		Build()
}

// submittedView overlays a refused submission and its error on v.
func submittedView(rc *collage.RenderContext, v formView) formView {
	if f, ok := Submitted.Get(rc); ok {
		v.Platform, v.Identifier, v.Passphrase, v.URL, v.Notes =
			f.Platform, f.Identifier, f.Passphrase, f.URL, f.Notes
	}
	v.Error = formError(rc)
	return v
}

// load fetches and decrypts account id, mapping a missing one to a 404.
func load(ctx context.Context, d *web.Deps, rc *collage.RenderContext, id string) (store.Account, error) {
	s, err := web.RequireSessionFrom(rc)
	if err != nil {
		return store.Account{}, err
	}
	defer s.DEK.Wipe()
	a, err := d.Store.GetAccount(ctx, s.DEK, id)
	if errors.Is(err, store.ErrNotFound) {
		return store.Account{}, fmt.Errorf("account %q: %w", id, collage.ErrNotFound)
	}
	return a, err
}
