// Package actions holds the generator's handler.
package actions

import (
	"context"
	"net/http"

	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages/generator"
	"passenger-go/internal/generator"
	"passenger-go/internal/web"
	"passenger-go/utils"
	"passenger-go/validations"
)

// Generate answers the generator page: a non-empty source is turned into its
// look-alike, otherwise a random passphrase of the given length is made. The
// answer is the page itself with the result; nothing is stored.
func Generate(_ *web.Deps) *collage.Action {
	return collage.NewAction("generate").
		WithMethods(http.MethodPost).
		WithHandler(generateHandler()).
		Build()
}

// generateHandler makes the look-alike or the random passphrase.
func generateHandler() collage.ActionHandlerFunc {
	return func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		v := utils.PostForm(rc)
		var (
			out string
			err error
		)
		if source := v.Value("source"); source != "" {
			out, err = generator.Alternate(source)
		} else {
			length := validations.Generate(v)
			if !v.Valid() {
				return validate.Refuse(rc, v, rc.Page), nil
			}
			out, err = generator.Generate(length.Value())
		}
		if err != nil {
			return nil, err
		}
		fragments.Generated.Set(rc, out)
		return collage.RenderPage(rc.Page), nil
	}
}
