package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/guards"
	"passenger-go/internal/web"
)

type recoverView struct {
	NewKey string
}

// Recover resets the master passphrase with the recovery key and, once it
// has, shows the replacement recovery key.
func Recover(d *web.Deps) *collage.Fragment {
	return collage.NewInlineFragment("recover", recoverBlock).
		WithTitle("Recover - Passenger").
		WithGuard(guards.Guest(d)).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (recoverView, error) {
			key, _ := NewRecoveryKey.Get(rc)
			return recoverView{NewKey: key}, nil
		})).
		Build()
}

const recoverBlock collage.InlineHTML = `{{if .NewKey}}
<section class="stack">
  <h1>Passphrase changed</h1>
  <p>This is your new recovery key. It is shown once: write it down and keep it safe. The previous key no longer works.</p>
  <div class="keycard">
    <code id="recovery-key">{{.NewKey}}</code>
    <button type="button" class="button" data-copy-input="recovery-key" data-name="Recovery key" hidden>Copy</button>
  </div>
  <a class="button button-primary button-wide" href="{{pageURL "login"}}">Continue to sign in</a>
</section>
{{else}}
<form method="post" action="{{pageURL "recover"}}" class="stack">
  {{csrfToken}}
  <h1>Recover your vault</h1>
  <p>Enter your recovery key and choose a new master passphrase. You will get a new recovery key, and the old one stops working.</p>

  <label class="field">
    <span>Recovery key</span>
    <input required name="recovery" autocomplete="off" spellcheck="false" class="code-input" autofocus>
  </label>

  <label class="field">
    <span>New master passphrase</span>
    <input required type="password" name="passphrase" minlength="12" autocomplete="new-password">
  </label>

  <label class="field">
    <span>Repeat the passphrase</span>
    <input required type="password" name="confirm" minlength="12" autocomplete="new-password">
  </label>

  {{if hasErrors}}<p class="alert alert-error" role="alert">{{or (fieldError "recovery") (fieldError "passphrase") (fieldError "confirm")}}</p>{{end}}

  <button type="submit" class="button button-primary button-wide">Recover vault</button>
</form>

<p class="pass-foot"><a href="{{pageURL "login"}}">Back to sign in</a></p>
{{end}}`
