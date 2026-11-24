package fragments

import "github.com/Elagoht/collage/pkg/collage"

// Export asks for the master passphrase before the CSV is downloaded.
func Export() *collage.Fragment {
	return collage.NewInlineFragment("export", exportBlock).
		WithTitle("Export - Passenger").
		Build()
}

const exportBlock collage.InlineHTML = `<div class="page-head">
  <a class="back" href="/">All accounts</a>
  <h1>Export</h1>
  <p>Download every account as a CSV you can import here or in a browser.</p>
</div>

<form method="post" action="{{pageURL "export"}}" class="panel stack">
  {{csrfToken}}
  <p class="alert alert-warn">The file holds every password as plain text. Keep it somewhere safe and delete it when you are done.</p>

  <label class="field">
    <span>Master passphrase</span>
    <input required type="password" name="passphrase" autocomplete="current-password">
  </label>

  {{with fieldError "passphrase"}}<p class="alert alert-error" role="alert">{{.}}</p>{{end}}

  <button type="submit" class="button button-primary">Download CSV</button>
</form>`
