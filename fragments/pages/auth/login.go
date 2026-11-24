package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"

	"passenger-go/guards"
	"passenger-go/internal/web"
	"passenger-go/utils"
)

type loginView struct {
	Next string
}

// Login is the unlock form. It carries next through the post in a hidden
// field.
func Login(d *web.Deps) *collage.Fragment {
	return collage.NewInlineFragment("login", loginBlock).
		WithTitle("Login - Passenger").
		WithGuard(guards.Guest(d)).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (loginView, error) {
			return loginView{Next: utils.SafeRedirect(rc.Request.FormValue("next"), "/")}, nil
		})).
		Build()
}

const loginBlock collage.InlineHTML = `<form method="post" action="{{pageURL "login"}}" class="stack">
  {{csrfToken}}
  <input type="hidden" name="next" value="{{.Next}}">
  <h1>Unlock your vault</h1>

  <label class="field">
    <span>Master passphrase</span>
    <input required type="password" name="passphrase" autocomplete="current-password" autofocus>
  </label>

  {{with fieldError "passphrase"}}<p class="alert alert-error" role="alert">{{.}}</p>{{end}}

  <button type="submit" class="button button-primary button-wide">Unlock</button>
</form>

<p class="pass-foot"><a href="{{pageURL "recover"}}">Forgot your passphrase?</a></p>`
