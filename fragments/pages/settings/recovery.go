package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"
)

type recoveryView struct {
	NewKey string
}

// Recovery issues a new recovery key.
func Recovery() *collage.Fragment {
	return collage.NewInlineFragment("settings-recovery", recoveryBlock).
		WithTitle("Recovery key - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (recoveryView, error) {
			key, _ := NewRecoveryKey.Get(rc)
			return recoveryView{NewKey: key}, nil
		})).
		Build()
}

const recoveryBlock collage.InlineHTML = `<div class="page-head">
  <a class="back" href="{{pageURL "settings-passphrase"}}">Settings</a>
  <h1>Recovery key</h1>
</div>

{{if .NewKey}}
<section class="panel stack">
  <p>This is your new recovery key. It is shown once: write it down and keep it safe. The previous key no longer works.</p>
  <div class="keycard">
    <code id="recovery-key">{{.NewKey}}</code>
    <button type="button" class="button" data-copy-input="recovery-key" data-name="Recovery key" hidden>Copy</button>
  </div>
  <a class="button button-primary" href="{{pageURL "settings-passphrase"}}">Done</a>
</section>
{{else}}
<form method="post" action="{{pageURL "settings-recovery"}}" autocomplete="off" class="panel stack">
  {{csrfToken}}
  <p>A new recovery key is shown once. The current key stops working right away.</p>

  <label class="field">
    <span>Master passphrase</span>
    <input required type="password" name="current" autocomplete="current-password">
  </label>

  {{with fieldError "current"}}<p class="alert alert-error" role="alert">{{.}}</p>{{end}}

  <button type="submit" class="button button-primary">Make a new recovery key</button>
</form>
{{end}}`
