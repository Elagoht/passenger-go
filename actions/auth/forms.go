// Package actions holds the form handlers of the setup, login, logout and
// recovery pages.
package actions

import (
	"net/http"
	"unicode/utf8"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages/auth"
)

// minPassphraseRunes is the shortest master passphrase accepted.
const minPassphraseRunes = 12

// checkNewPassphrase returns why a new master passphrase and its repeat are
// refused, or "" when they are accepted.
func checkNewPassphrase(pass, confirm string) string {
	if utf8.RuneCountInString(pass) < minPassphraseRunes {
		return "The passphrase must be at least 12 characters."
	}
	if pass != confirm {
		return "The passphrases do not match."
	}
	return ""
}

// refuse answers with the form's own page carrying msg. The page renders no
// submitted secret back.
func refuse(rc *collage.RenderContext, status int, msg string) *collage.ActionResult {
	fragments.FormError.Set(rc, msg)
	return &collage.ActionResult{Status: status, Page: rc.Page}
}

// throttled is the 429 answer for a client backing off.
func throttled(rc *collage.RenderContext, msg string) *collage.ActionResult {
	return refuse(rc, http.StatusTooManyRequests, msg)
}
