package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/guards"
	"passenger-go/internal/web"
)

type setupConfirmView struct {
	Recovery string
}

// SetupConfirm shows the pending setup's recovery key and asks for it back.
// Its guard turns away requests without a live pending setup.
func SetupConfirm(d *web.Deps) *collage.Fragment {
	return collage.NewInlineFragment("setup-confirm", setupConfirmBlock).
		WithTitle("Recovery key - Passenger").
		WithGuard(guards.PendingSetup(d)).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (setupConfirmView, error) {
			_, recovery, dek, _ := d.Pending.Get(web.PendingToken(rc.Request))
			dek.Wipe()
			return setupConfirmView{Recovery: recovery}, nil
		})).
		Build()
}

const setupConfirmBlock collage.InlineHTML = `<form method="post" action="{{pageURL "setup-confirm"}}" class="stack">
  {{csrfToken}}
  <h1>Save your recovery key</h1>
  <p>If you forget your master passphrase, this key is the only way back in. It is shown once, so write it down or store it somewhere safe now.</p>

  <div class="keycard">
    <code id="recovery-key">{{.Recovery}}</code>
    <button type="button" class="button" data-copy-input="recovery-key" data-name="Recovery key" hidden>Copy</button>
  </div>

  <label class="field">
    <span>Type the key to confirm you saved it</span>
    <input required name="recovery" autocomplete="off" spellcheck="false" class="code-input">
  </label>

  {{with fieldError "recovery"}}<p class="alert alert-error" role="alert">{{.}}</p>{{end}}

  <button type="submit" class="button button-primary button-wide">Create vault</button>
</form>`
