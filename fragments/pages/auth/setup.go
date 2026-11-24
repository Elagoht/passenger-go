package fragments

import (
	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/guards"
	"passenger-go/internal/web"
)

// Setup is the first-run form: setup code, master passphrase and its repeat.
// It is reachable only while no vault exists.
func Setup(d *web.Deps) *collage.Fragment {
	return collage.NewInlineFragment("setup", setupBlock).
		WithTitle("Setup - Passenger").
		WithGuard(guards.Uninitialized(d)).
		Build()
}

const setupBlock collage.InlineHTML = `<form method="post" action="{{pageURL "setup"}}" class="stack">
  {{csrfToken}}
  <h1>Create your vault</h1>
  <p>The setup code is in the server log, on the line that starts with “setup code”. Use the newest one.</p>

  <label class="field">
    <span>Setup code</span>
    <input required name="code" inputmode="numeric" autocomplete="off" maxlength="8" class="code-input" autofocus>
  </label>

  <label class="field">
    <span>Master passphrase</span>
    <input required type="password" name="passphrase" minlength="12" autocomplete="new-password">
    <small>At least 12 characters. This is the only key to your vault.</small>
  </label>

  <label class="field">
    <span>Repeat the passphrase</span>
    <input required type="password" name="confirm" minlength="12" autocomplete="new-password">
  </label>

  {{if hasErrors}}<p class="alert alert-error" role="alert">{{or (fieldError "code") (fieldError "passphrase") (fieldError "confirm")}}</p>{{end}}

  <button type="submit" class="button button-primary button-wide">Continue</button>
</form>`
