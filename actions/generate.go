// Package actions holds the generator's handler.
package actions

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages"
	"passenger-go/internal/generator"
	"passenger-go/internal/web"
)

// Generate answers the generator page: a non-empty source is turned into its
// look-alike, otherwise a random passphrase of the given length is made. The
// answer is the page itself with the result; nothing is stored.
func Generate(_ *web.Deps) *collage.Action {
	return collage.NewAction("generate").
		WithMethods(http.MethodPost).
		WithHandler(func(_ context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
			r := rc.Request
			var (
				out string
				err error
			)
			if source := r.PostFormValue("source"); source != "" {
				out, err = generator.Alternate(source)
			} else {
				n, perr := strconv.Atoi(strings.TrimSpace(r.PostFormValue("length")))
				if perr != nil {
					fragments.FormError.Set(rc, "Length must be a number.")
					return &collage.ActionResult{Status: http.StatusUnprocessableEntity, Page: rc.Page}, nil
				}
				out, err = generator.Generate(n)
			}
			if err != nil {
				return nil, err
			}
			fragments.Generated.Set(rc, out)
			return collage.RenderPage(rc.Page), nil
		}).
		Build()
}
