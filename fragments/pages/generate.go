// Package fragments holds the content of the generator page.
package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"
)

// Generated is what the generator produced, shown once on the page that
// answers the request.
var Generated = collage.NewKey[string]("generate-result")

// FormError is why the generator refused a request.
var FormError = collage.NewKey[string]("generate-form-error")

// DefaultLength is the length the form starts with.
const DefaultLength = 24

type generateView struct {
	Result string
	Error  string
	Length int
}

// Generate is the passphrase generator and the look-alike "alternator".
func Generate() *collage.Fragment {
	return collage.NewFragment("generate", "pages/generate.html").
		WithTitle("Generate - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (generateView, error) {
			res, _ := Generated.Get(rc)
			msg, _ := FormError.Get(rc)
			return generateView{Result: res, Error: msg, Length: DefaultLength}, nil
		})).
		Build()
}
