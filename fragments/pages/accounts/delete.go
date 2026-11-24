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
	return collage.NewInlineFragment("account-delete", deleteBlock).
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

const deleteBlock collage.InlineHTML = `<section class="confirm">
  <h1>Delete {{.Platform}}?</h1>
  <p>The account for <strong>{{.Identifier}}</strong> and its password are removed from the vault. This cannot be undone.</p>
  <form method="post" action="{{pageURL "account-delete" "id" .ID}}" class="actions">
    {{csrfToken}}
    <button type="submit" class="button button-danger">Delete account</button>
    <a class="button button-quiet" href="{{pageURL "account" "id" .ID}}">Keep it</a>
  </form>
</section>`
