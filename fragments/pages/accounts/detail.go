package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/web"
)

// Detail shows one account for editing, its passphrase masked. An unknown id
// is a 404.
func Detail(d *web.Deps) *collage.Fragment {
	return collage.NewInlineFragment("account-detail", formBlock).
		WithTitle("Account - Passenger").
		Required().
		WithData(collage.Load(func(ctx context.Context, rc *collage.RenderContext) (formView, error) {
			a, err := load(ctx, d, rc, rc.Param("id"))
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
