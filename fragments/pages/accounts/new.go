package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"
)

// New is the empty account form. Reachable only behind a session.
func New() *collage.Fragment {
	return collage.NewInlineFragment("account-new", formBlock).
		WithTitle("New account - Passenger").
		WithData(collage.Load(func(_ context.Context, _ *collage.RenderContext) (formView, error) {
			return formView{}, nil
		})).
		Build()
}
