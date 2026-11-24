// Package fragments holds the content of the generator page.
package fragments

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Elagoht/collage/pkg/collage"
)

// Generated is what the generator produced, shown once in the result.
var Generated = collage.NewKey[string]("generate-result")

// GenerateError is why a request was refused, shown in the result.
var GenerateError = collage.NewKey[string]("generate-error")

// DefaultLength is the length the form starts with.
const DefaultLength = 24

type generateView struct {
	Length int
	Source string
}

type resultView struct {
	Result string
	Error  string
}

// Result is the part of the generator page that changes: the generated
// passphrase or the reason it was refused. The generator's forms post with
// fetch and the action answers with this fragment alone, which the page puts
// in place, so the forms and what is typed in them stay as they are.
func Result() *collage.Fragment {
	return collage.NewInlineFragment("generate-result", resultBlock).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (resultView, error) {
			res, _ := Generated.Get(rc)
			msg, _ := GenerateError.Get(rc)
			return resultView{Result: res, Error: msg}, nil
		})).
		Build()
}

// Generate is the passphrase generator and the look-alike "alternator",
// with result in its result slot. Without JavaScript the forms post as usual
// and the whole page answers, keeping what was typed.
func Generate(result *collage.Fragment) *collage.Fragment {
	return collage.NewInlineFragment("generate", generateBlock).
		WithTitle("Generate - Passenger").
		WithSlotFragment("result", result).
		WithData(collage.Load(func(_ context.Context, rc *collage.RenderContext) (generateView, error) {
			view := generateView{Length: DefaultLength}
			if rc.Request.Method == http.MethodPost {
				view.Source = rc.Request.PostFormValue("source")
				if n, err := strconv.Atoi(rc.Request.PostFormValue("length")); err == nil {
					view.Length = n
				}
			}
			return view, nil
		})).
		Build()
}

const resultBlock collage.InlineHTML = `<div id="generator-result" aria-live="polite">
  {{with .Error}}<p class="alert alert-error" role="alert">{{.}}</p>{{end}}
  {{with .Result}}
  <div class="keycard keycard-result">
    <input type="text" id="generated" readonly value="{{.}}" class="secret-input" aria-label="Result">
    <button type="button" class="button button-primary" data-copy-input="generated" data-name="Password" hidden>Copy</button>
  </div>
  {{end}}
</div>`

const generateBlock collage.InlineHTML = `<div class="page-head">
  <h1>Generator</h1>
  <p>Make a new random password, or turn a word you can remember into a harder one.</p>
</div>

{{slot "result"}}

<div class="panels">
  <form method="post" action="{{pageURL "generate"}}" class="panel stack" data-live="#generator-result">
    {{csrfToken}}
    <h2>Random password</h2>
    <label class="field">
      <span>Length</span>
      <input type="number" name="length" min="8" max="4096" value="{{.Length}}">
      <small>Always mixes lower and upper case letters, digits and symbols.</small>
    </label>
    <button type="submit" class="button button-primary">Generate</button>
  </form>

  <form method="post" action="{{pageURL "generate"}}" class="panel stack" data-live="#generator-result">
    {{csrfToken}}
    <h2>Look-alike version</h2>
    <label class="field">
      <span>Word or phrase</span>
      <input type="text" name="source" value="{{.Source}}" autocomplete="off" spellcheck="false">
      <small>Swaps letters for look-alikes, so <code>password</code> can become <code>p@55w0rd</code>.</small>
    </label>
    <button type="submit" class="button">Make look-alike</button>
  </form>
</div>`
