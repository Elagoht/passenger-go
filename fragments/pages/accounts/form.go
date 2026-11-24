// Package fragments holds the content of the account pages.
package fragments

import (
	"context"
	"errors"
	"fmt"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/internal/store"
	"passenger-go/internal/web"
)

// formView is the account form's data, shared by New and Detail: the stored
// account, or nothing for a new one. A refused submission is typed back over
// it by {{fieldValue}}, the account's passphrase included (only the master
// passphrase and the recovery key are never echoed).
type formView struct {
	ID         string // empty for a new account
	Platform   string
	Identifier string
	Passphrase string
	URL        string
	Notes      string
}

const formBlock collage.InlineHTML = `<div class="page-head">
  <a class="back" href="/">All accounts</a>
  <h1>{{if .ID}}{{if .Platform}}{{.Platform}}{{else}}Account{{end}}{{else}}New account{{end}}</h1>
</div>

<form class="account-form" method="post" action="{{if .ID}}{{pageURL "account" "id" .ID}}{{else}}{{pageURL "account-new"}}{{end}}" autocomplete="off" data-track>
  {{csrfToken}}

  <div class="fields">
    <label class="field">
      <span>Site or app</span>
      <input required type="text" name="platform" value="{{fieldValue "platform" .Platform}}" placeholder="GitHub"{{if not .ID}} autofocus{{end}}>
    </label>

    <label class="field">
      <span>Username or email</span>
      <input required type="text" name="identifier" value="{{fieldValue "identifier" .Identifier}}" spellcheck="false">
    </label>

    <label class="field">
      <span>Address</span>
      <input type="text" inputmode="url" name="url" value="{{fieldValue "url" .URL}}" placeholder="github.com" spellcheck="false">
    </label>

    <label class="field">
      <span>Notes</span>
      <textarea name="notes" rows="4">{{fieldValue "notes" .Notes}}</textarea>
    </label>
  </div>

  <div class="secret">
    <label class="field" for="passphrase"><span>Password</span></label>
    <input required type="password" id="passphrase" name="passphrase" value="{{fieldValue "passphrase" .Passphrase}}" autocomplete="new-password" spellcheck="false" class="secret-input">
    <div class="secret-actions">
      <button type="button" class="button" data-reveal="passphrase" hidden>Show</button>
      <button type="button" class="button" data-copy-input="passphrase" data-name="Password" hidden>Copy</button>
      <button type="button" class="button" data-generate-into="passphrase" hidden>Generate</button>
    </div>
    <p class="secret-meter" data-meter-for="passphrase" aria-live="polite"></p>
  </div>

  {{if hasErrors}}<p class="alert alert-error" role="alert">{{or (fieldError "platform") (fieldError "identifier") (fieldError "passphrase")}}</p>{{end}}

  <div class="form-foot">
    <button type="submit" class="button button-primary">{{if .ID}}Save changes{{else}}Add account{{end}}</button>
    <a class="button button-quiet" href="/">Cancel</a>
    {{if .ID}}<a class="button button-danger-quiet" href="{{pageURL "account-delete" "id" .ID}}">Delete account</a>{{end}}
  </div>
</form>`

// load fetches and decrypts account id, mapping a missing one to a 404.
func load(ctx context.Context, d *web.Deps, rc *collage.RenderContext, id string) (store.Account, error) {
	s, err := web.RequireSessionFrom(rc)
	if err != nil {
		return store.Account{}, err
	}
	defer s.DEK.Wipe()
	a, err := d.Store.GetAccount(ctx, s.DEK, id)
	if errors.Is(err, store.ErrNotFound) {
		return store.Account{}, fmt.Errorf("account %q: %w", id, collage.ErrNotFound)
	}
	return a, err
}
