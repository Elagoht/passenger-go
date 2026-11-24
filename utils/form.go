package utils

import (
	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"
)

// PostForm is a validator over the request's body alone, for a form that
// sends a secret: validate.Form reads the query too, as a fallback for a field
// the body lacks, and a passphrase is never taken from a URL. A body that
// cannot be parsed leaves the fields empty, which the checks then refuse.
func PostForm(rc *collage.RenderContext) *validate.Validator {
	_ = rc.Request.ParseForm()
	return validate.Values(rc, rc.Request.PostForm)
}
