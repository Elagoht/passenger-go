package actions

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/guards"
	"passenger-go/internal/store"
	"passenger-go/internal/web"
)

type passphraseResponse struct {
	Passphrase string `json:"passphrase"`
}

// Passphrase answers the list's "copy password" button with account {id}'s
// passphrase as JSON. It has a URL of its own, so it carries its own guard;
// the forgery check wants the token in X-CSRF-Token.
func Passphrase(d *web.Deps) *collage.Action {
	return collage.NewAction("account-passphrase").
		WithPath("en", "/accounts/{id}/passphrase").
		WithMethods(http.MethodPost).
		WithGuard(guards.SignedIn(d)).
		WithHandler(passphraseHandler(d)).
		Build()
}

// passphraseHandler decrypts the account and answers with its passphrase; an
// unknown one is a 404.
func passphraseHandler(d *web.Deps) collage.ActionHandlerFunc {
	return func(ctx context.Context, rc *collage.RenderContext) (*collage.ActionResult, error) {
		id := rc.Param("id")
		s, err := web.RequireSessionFrom(rc)
		if err != nil {
			return nil, err
		}
		defer s.DEK.Wipe()
		a, err := d.Store.GetAccount(ctx, s.DEK, id)
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("account %q: %w", id, collage.ErrNotFound)
		}
		if err != nil {
			return nil, err
		}
		return collage.JSON(http.StatusOK, passphraseResponse{Passphrase: a.Passphrase})
	}
}
