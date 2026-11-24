package fragments

import "github.com/Elagoht/collage/pkg/collage"

// Passphrase changes the master passphrase. "Passphrase changed." comes back
// after the redirect as a flash message, shown by the App layout.
func Passphrase() *collage.Fragment {
	return collage.NewInlineFragment("settings-passphrase", passphraseBlock).
		WithTitle("Settings - Passenger").
		Build()
}

const passphraseBlock collage.InlineHTML = `<div class="page-head">
  <h1>Settings</h1>
</div>

<div class="panels">
  <form method="post" action="{{pageURL "settings-passphrase"}}" autocomplete="off" class="panel stack">
    {{csrfToken}}
    <h2>Master passphrase</h2>
    <p>Changing it signs out every other device.</p>

    <label class="field">
      <span>Current passphrase</span>
      <input required type="password" name="current" autocomplete="current-password">
    </label>

    <label class="field">
      <span>New passphrase</span>
      <input required type="password" name="passphrase" minlength="12" autocomplete="new-password">
    </label>

    <label class="field">
      <span>Repeat the new passphrase</span>
      <input required type="password" name="confirm" minlength="12" autocomplete="new-password">
    </label>

    {{if hasErrors}}<p class="alert alert-error" role="alert">{{or (fieldError "current") (fieldError "passphrase") (fieldError "confirm")}}</p>{{end}}

    <button type="submit" class="button button-primary">Change passphrase</button>
  </form>

  <section class="panel stack">
    <h2>Recovery key</h2>
    <p>Lost your recovery key, or think someone else has seen it? Make a new one; the current key stops working.</p>
    <a class="button" href="{{pageURL "settings-recovery"}}">Regenerate recovery key</a>
  </section>
</div>`
