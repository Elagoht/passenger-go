package utils

import (
	"net/http"

	validate "github.com/Elagoht/collage-validate"
	"github.com/Elagoht/collage/pkg/collage"
)

// Refuse answers a refused submission with its own page, 422, the reason on
// field: what the application found wrong (a wrong passphrase, a duplicate)
// beside what the checks in validations/ found.
func Refuse(rc *collage.RenderContext, v *validate.Validator, field, msg string) *collage.ActionResult {
	v.Fail(field, msg)
	return validate.Refuse(rc, v, rc.Page)
}

// Throttled is Refuse for a client backing off from guessing a secret: the
// same page, 429.
func Throttled(rc *collage.RenderContext, v *validate.Validator, field, msg string) *collage.ActionResult {
	res := Refuse(rc, v, field, msg)
	res.Status = http.StatusTooManyRequests
	return res
}
