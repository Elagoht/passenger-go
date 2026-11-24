// Package actions holds the handlers behind the account pages.
package actions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Elagoht/collage/pkg/collage"

	fragments "passenger-go/fragments/pages/accounts"
	"passenger-go/internal/store"
	"passenger-go/internal/vault"
)

const msgDuplicate = "An account with this platform and identifier already exists."

// readForm returns what the account form posted. Platform, identifier and URL
// are trimmed; the passphrase and notes are taken as typed.
func readForm(r *http.Request) fragments.Form {
	return fragments.Form{
		Platform:   strings.TrimSpace(r.PostFormValue("platform")),
		Identifier: strings.TrimSpace(r.PostFormValue("identifier")),
		Passphrase: r.PostFormValue("passphrase"),
		URL:        strings.TrimSpace(r.PostFormValue("url")),
		Notes:      r.PostFormValue("notes"),
	}
}

// problem says what is missing from f, or "" when it can be stored.
func problem(f fragments.Form) string {
	switch {
	case f.Platform == "":
		return "Platform is required."
	case f.Identifier == "":
		return "Identifier is required."
	case f.Passphrase == "":
		return "Passphrase is required."
	}
	return ""
}

// refuse re-renders the form's own page (422) with what was typed and why it
// was refused.
func refuse(rc *collage.RenderContext, f fragments.Form, msg string) *collage.ActionResult {
	fragments.Submitted.Set(rc, f)
	fragments.FormError.Set(rc, msg)
	return &collage.ActionResult{Status: http.StatusUnprocessableEntity, Page: rc.Page}
}

func record(f fragments.Form) vault.Record {
	return vault.Record{
		Platform: f.Platform, Identifier: f.Identifier, Passphrase: f.Passphrase,
		URL: f.URL, Notes: f.Notes,
	}
}

func isDuplicate(err error) bool { return errors.Is(err, store.ErrDuplicate) }
