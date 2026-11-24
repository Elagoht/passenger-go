// Package fragments holds the content of the generator page.
package fragments

import (
	"context"

	"github.com/Elagoht/collage/pkg/collage"
)

// Generated is what the generator produced, shown once on the page that
// answers the request.
var Generated = collage.NewKey[string]("generate-result")

// DefaultLength is the length the form starts with.
const DefaultLength = 24

type generateView struct {
	Result string
	Length int
}

// Generate is the passphrase generator and the look-alike "alternator".
func Generate() *collage.Fragment {
	return collage.NewInlineFragment("generate", generateBlock).
		WithTitle("Generate - Passenger").
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (generateView, error) {
			res, _ := Generated.Get(rc)
			return generateView{Result: res, Length: DefaultLength}, nil
		})).
		Build()
}

const generateBlock collage.InlineHTML = `<div class="page-head">
  <h1>Generator</h1>
  <p>Make a new random password, or turn a word you can remember into a harder one.</p>
</div>

{{with .Result}}
<div class="keycard keycard-result">
  <input type="text" id="generated" readonly value="{{.}}" class="secret-input" aria-label="Result">
  <button type="button" class="button button-primary" data-copy-input="generated" data-name="Password" hidden>Copy</button>
</div>
{{end}}

{{with fieldError "length"}}<p class="alert alert-error" role="alert">{{.}}</p>{{end}}

<div class="panels">
  <form method="post" action="{{pageURL "generate"}}" class="panel stack">
    {{csrfToken}}
    <h2>Random password</h2>
    <label class="field">
      <span>Length</span>
      <input type="number" name="length" min="8" max="4096" value="{{.Length}}">
      <small>Always mixes lower and upper case letters, digits and symbols.</small>
    </label>
    <button type="submit" class="button button-primary">Generate</button>
  </form>

  <form method="post" action="{{pageURL "generate"}}" class="panel stack">
    {{csrfToken}}
    <h2>Look-alike version</h2>
    <label class="field">
      <span>Word or phrase</span>
      <input type="text" name="source" autocomplete="off" spellcheck="false">
      <small>Swaps letters for look-alikes, so <code>password</code> can become <code>p@55w0rd</code>.</small>
    </label>
    <button type="submit" class="button">Make look-alike</button>
  </form>
</div>`
