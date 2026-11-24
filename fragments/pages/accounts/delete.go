package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

type deleteView struct {
	ID         string
	Platform   string
	Identifier string
}

// Delete asks for confirmation before an account is removed; the POST to the
// same URL deletes. An unknown id is a 404.
func Delete(d *web.Deps) *collage.Fragment {
	return collage.NewFragment("account-delete", "pages/accounts/delete.html").
		WithTitle("Delete account - Passenger").
		Required().
		WithData(collage.Load(func(ctx context.Context, rc *collage.RenderContext) (deleteView, error) {
			a, err := load(ctx, d, rc, rc.Param("id"))
			if err != nil {
				return deleteView{}, err
			}
			return deleteView{ID: a.ID, Platform: a.Platform, Identifier: a.Identifier}, nil
		})).
		Build()
}
